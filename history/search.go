// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"cmp"
	"slices"
	"sync/atomic"
	"time"

	"go.dokimi.dev/assert/internal/fault"
)

// clockSteps is the number of steps between two readings of the deadline.
const clockSteps = 1024

// keyPrime is the odd multiplier that mixes each word of a set of calls into
// the key of a configuration: the 64-bit FNV prime.
const keyPrime = 1099511628211

// The model's functions, as the fault of one that panics names it.
const (
	initFunction  = "Init"
	stepFunction  = "Step"
	equalFunction = "Equal"
	hashFunction  = "Hash"
)

// deadline is the time limit of one check, which every search of the check
// reads, and the checker's cancellation of the searches that it no longer
// needs.
type deadline struct {
	// end is when the time limit passes, on the platform clock, and the zero
	// time for a check without one.
	end time.Time
	// cancelled reports whether the checker has cancelled the searches.
	cancelled atomic.Bool
}

// passed reports whether a search of the check stops: the checker cancelled
// it, or the time limit has passed.
func (d *deadline) passed() bool {
	return d.cancelled.Load() || (!d.end.IsZero() && time.Now().After(d.end))
}

// ending is how the search of one partition ended: its verdict, its steps
// and its frontier, or the fault of one of the model's functions.
type ending[S any] struct {
	// verdict is the partition's verdict.
	verdict Verdict
	// steps is the steps that the search spent.
	steps int
	// linearized are the positions of the frontier's calls, in order.
	linearized []int
	// states are the frontier's states.
	states []S
	// candidates are the positions of the calls that the model rejected at
	// the frontier.
	candidates []int
	// limit is the limit that stopped an undecided search.
	limit Limit
	// err is the fault of a model's function that panicked or ended the
	// goroutine, and nil for a search that ended.
	err error
}

// extent is a list of states in the store of a search.
type extent struct {
	// at is the index of the first state in the store.
	at int
	// n is the number of states.
	n int
}

// frame is a linearized call, and the states before it.
type frame struct {
	// position is the call's position in the partition.
	position int
	// states are the states before the call.
	states extent
}

// configuration is a configuration in the memo: a set of linearized calls
// and the states that they leave, no two of them equal, and the next
// configuration of its key. The set's further words, one for each 64 calls
// after the first 64, are in the search's words, at the configuration's
// index times their number.
type configuration struct {
	// first is the first word of the set of calls, a bit for each of the
	// first 64 calls of the partition.
	first uint64
	// states are the states in the store.
	states extent
	// next is the index of the next configuration of the same key, and -1
	// for the last.
	next int
}

// search is the search of one partition, which the definition fixes: Wing
// and Gong's scan of the entries, with a memo of the configurations that it
// visited. The entries are the invocations of the partition's calls and the
// completions of its known calls, in event order, in a doubly linked list
// between a head and a tail. The search takes a linearized call's entries
// out of the scan and puts them back in constant time.
//
// The states of every configuration are in one store, and a configuration
// and a frame refer to their states through an extent of it. A step leaves
// its states in a buffer, and the search copies them to the store only for a
// configuration that the memo lacks.
type search[S any] struct {
	// ops are the model's functions, with the defaults of Equal and Hash.
	ops operations[S]
	// calls are the partition's calls, in event order.
	calls []call
	// budget is the steps that the search may spend.
	budget int
	// capacity is the most configurations that the memo may store: the memo
	// limit divided by the number of calls.
	capacity int64
	// deadline is the check's deadline. The search reads it before its first
	// step and every clockSteps steps after it.
	deadline *deadline
	// clockAt is the number of steps at which the search reads the deadline
	// next.
	clockAt int
	// final reports whether a search that passes states the states that its
	// order leaves.
	final bool

	// next and prev link each entry to the entries beside it. The head is the
	// entry len(position), and the tail the one after the head.
	next, prev []int
	// position is the position of the call of each entry.
	position []int
	// invokes reports whether each entry is an invocation.
	invokes []bool
	// invocation and completion are the entries of each call's invocation
	// and of a known call's completion.
	invocation, completion []int
	// open is the number of known calls that are not linearized.
	open int

	// steps is the steps spent so far.
	steps int
	// done is the set of linearized calls, a bit for each call's position.
	done []uint64
	// states are the states that the linearized calls leave.
	states extent
	// stack are the linearized calls, in the order linearized.
	stack []frame
	// store are the states of the configurations, the initial state first.
	store []S
	// words are the further words of the sets of calls of the configurations,
	// in the order of the configurations.
	words []uint64
	// memo maps the key of each configuration that the search visited to the
	// index of the newest configuration of that key.
	memo map[uint64]int
	// configurations are the configurations in the memo.
	configurations []configuration

	// scratch are the states that the step of a call leaves, and hashes the
	// hash of each.
	scratch []S
	hashes  []uint64

	// linearized are the positions of the frontier's calls, in order.
	linearized []int
	// frontier are the frontier's states.
	frontier extent
	// candidates are the positions of the calls that the model rejected at
	// the frontier so far.
	candidates []int
	// atFrontier reports whether the current configuration is the frontier.
	atFrontier bool

	// calling is the model's function that the search calls.
	calling string
	// stepped is the position of the call that the search steps, and -1
	// before the first step.
	stepped int
}

// newSearch returns the search of p through ops under limits and the
// deadline d.
func newSearch[S any](ops operations[S], p partition, limits config, d *deadline) *search[S] {
	type entry struct {
		// event is the index of the entry's event.
		event int
		// position is the position of the entry's call.
		position int
		// invokes reports whether the entry is an invocation.
		invokes bool
	}
	var entries []entry
	open := 0
	for position, c := range p.calls {
		entries = append(entries, entry{event: c.span.Call, position: position, invokes: true})
		if c.span.Op.Known {
			entries = append(entries, entry{event: c.span.Completion, position: position})
			open++
		}
	}
	slices.SortFunc(entries, func(a, b entry) int { return cmp.Compare(a.event, b.event) })

	head := len(entries)
	s := &search[S]{
		ops:        ops,
		calls:      p.calls,
		budget:     limits.budget,
		capacity:   limits.memoLimit / int64(max(len(p.calls), 1)),
		deadline:   d,
		final:      limits.final != nil,
		next:       make([]int, head+2),
		prev:       make([]int, head+2),
		position:   make([]int, head),
		invokes:    make([]bool, head),
		invocation: make([]int, len(p.calls)),
		completion: make([]int, len(p.calls)),
		open:       open,
		done:       make([]uint64, (len(p.calls)+63)>>6),
		memo:       map[uint64]int{},
		atFrontier: true,
		stepped:    -1,
	}
	before := head
	for i, e := range entries {
		s.next[before], s.prev[i] = i, before
		before = i
		s.position[i], s.invokes[i] = e.position, e.invokes
		if e.invokes {
			s.invocation[e.position] = i
		} else {
			s.completion[e.position] = i
		}
	}
	s.next[before], s.prev[head+1] = head+1, before
	return s
}

// run searches the partition and passes how the search ended to deliver, on
// every path out of it: the verdict and the frontier, or the fault of a
// model's function that panicked or ended the goroutine. After a function
// that ends the goroutine, as t.FailNow does, the goroutine ends once
// deliver returns.
func (s *search[S]) run(deliver func(ending[S])) {
	var e ending[S]
	ended := false
	defer func() {
		if v := recover(); v != nil || !ended {
			e = ending[S]{err: s.fault(v)}
		}
		deliver(e)
	}()
	s.calling = initFunction
	s.store = append(s.store, s.ops.init())
	s.states = extent{n: 1}
	s.frontier = s.states
	e = s.scan()
	ended = true
}

// scan scans the entries until every known call is linearized, or no order
// that the model accepts is left, or a limit stops it.
func (s *search[S]) scan() ending[S] {
	head := len(s.position)
	entry := s.next[head]
	for s.open > 0 {
		if !s.invokes[entry] {
			if len(s.stack) == 0 {
				return s.ending(Violated, 0)
			}
			entry = s.backtrack()
			continue
		}
		accepted, limit := s.linearize(s.position[entry])
		if limit != 0 {
			return s.ending(Undecided, limit)
		}
		if accepted {
			entry = s.next[head]
		} else {
			entry = s.next[entry]
		}
	}
	e := ending[S]{verdict: Passed, steps: s.steps}
	if s.final {
		e.states = slices.Clone(s.store[s.states.at : s.states.at+s.states.n])
	}
	return e
}

// ending returns the ending of a search that did not pass: v and limit, the
// steps, and the frontier, whose states it copies out of the store.
func (s *search[S]) ending(v Verdict, limit Limit) ending[S] {
	return ending[S]{
		verdict:    v,
		steps:      s.steps,
		linearized: s.linearized,
		states:     slices.Clone(s.store[s.frontier.at : s.frontier.at+s.frontier.n]),
		candidates: s.candidates,
		limit:      limit,
	}
}

// linearize linearizes the call at position when the model accepts it and
// the configuration that it leads to is new. It reports whether it did, and
// the limit that stopped the search.
func (s *search[S]) linearize(position int) (bool, Limit) {
	s.stepped = position
	sum, limit := s.step(s.calls[position].span.Op)
	if limit != 0 {
		return false, limit
	}
	if len(s.scratch) == 0 {
		if s.atFrontier {
			s.candidates = append(s.candidates, position)
		}
		return false, 0
	}
	word, bit := position>>6, uint64(1)<<(position&63)
	s.done[word] |= bit
	states, stored, limit := s.remember(sum)
	if !stored {
		s.done[word] &^= bit
		return false, limit
	}
	s.stack = append(s.stack, frame{position: position, states: s.states})
	s.states = states
	s.lift(position)
	s.atFrontier = len(s.stack) > len(s.linearized)
	if s.atFrontier {
		s.linearized = s.linearized[:0]
		for _, f := range s.stack {
			s.linearized = append(s.linearized, f.position)
		}
		s.frontier, s.candidates = states, s.candidates[:0]
	}
	return true, 0
}

// step leaves in scratch the states that op leaves from the current states,
// without two equal states, with their hashes in hashes, and returns the sum
// of the hashes. It reports the limit that stops the search before a step:
// the budget, or the deadline.
func (s *search[S]) step(op Op) (uint64, Limit) {
	s.scratch, s.hashes = s.scratch[:0], s.hashes[:0]
	var sum uint64
	current := s.store[s.states.at : s.states.at+s.states.n]
	for i := range current {
		cost := 1
		if s.ops.cost != nil {
			cost = s.ops.cost(current[i])
		}
		if s.steps+cost > s.budget {
			return 0, LimitSteps
		}
		if s.steps >= s.clockAt {
			if s.deadline.passed() {
				return 0, LimitTime
			}
			s.clockAt = s.steps + clockSteps
		}
		s.steps += cost
		s.calling = stepFunction
		next := s.ops.step(current[i], op)
		for j := range next {
			var hash uint64
			if s.ops.hash != nil {
				s.calling = hashFunction
				hash = s.ops.hash(&next[j])
			}
			if s.left(&next[j], hash) {
				continue
			}
			s.scratch = append(s.scratch, next[j])
			s.hashes = append(s.hashes, hash)
			sum += hash
		}
	}
	return sum, 0
}

// left reports whether scratch contains a state equal to the state at state,
// whose hash is hash. Two states of different hashes are different states.
func (s *search[S]) left(state *S, hash uint64) bool {
	s.calling = equalFunction
	for i := range s.scratch {
		if s.hashes[i] == hash && s.ops.equal(state, &s.scratch[i]) {
			return true
		}
	}
	return false
}

// remember stores the configuration of the linearized calls and of the
// states in scratch, whose hashes sum to sum, unless the memo has it
// already. It returns the states in the store and reports whether it stored
// the configuration, and the limit that stops the search before a
// configuration that would pass the memo limit.
func (s *search[S]) remember(sum uint64) (extent, bool, Limit) {
	key := sum
	for _, word := range s.done {
		key = (key ^ word) * keyPrime
	}
	head, found := s.memo[key]
	if !found {
		head = -1
	}
	rest := s.done[1:]
	for i := head; i != -1; i = s.configurations[i].next {
		c := s.configurations[i]
		if c.first == s.done[0] && slices.Equal(s.words[i*len(rest):i*len(rest)+len(rest)], rest) &&
			s.same(c.states) {
			return extent{}, false, 0
		}
	}
	if int64(len(s.configurations)) >= s.capacity {
		return extent{}, false, LimitMemo
	}
	states := extent{at: len(s.store), n: len(s.scratch)}
	s.memo[key] = len(s.configurations)
	s.configurations = append(s.configurations, configuration{first: s.done[0], states: states, next: head})
	s.words = append(s.words, rest...)
	s.store = append(s.store, s.scratch...)
	return states, true, 0
}

// same reports whether the states at seen, a configuration's in the memo,
// are the states in scratch: each of scratch equals one of seen, and the two
// have one length. Neither lists two equal states, so their order does not
// matter.
func (s *search[S]) same(seen extent) bool {
	if seen.n != len(s.scratch) {
		return false
	}
	kept := s.store[seen.at : seen.at+seen.n]
	for i := range s.scratch {
		if !s.contains(kept, &s.scratch[i]) {
			return false
		}
	}
	return true
}

// contains reports whether states contains a state equal to the state at
// state.
func (s *search[S]) contains(states []S, state *S) bool {
	s.calling = equalFunction
	for i := range states {
		if s.ops.equal(state, &states[i]) {
			return true
		}
	}
	return false
}

// backtrack undoes the last linearized call, and returns the entry after its
// invocation.
func (s *search[S]) backtrack() int {
	top := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	s.states = top.states
	s.done[top.position>>6] &^= uint64(1) << (top.position & 63)
	s.unlift(top.position)
	s.atFrontier = false
	return s.next[s.invocation[top.position]]
}

// lift takes the entries of the call at position out of the scan.
func (s *search[S]) lift(position int) {
	s.unlink(s.invocation[position])
	if s.calls[position].span.Op.Known {
		s.unlink(s.completion[position])
		s.open--
	}
}

// unlift puts the entries of the call at position back, in the reverse order
// of lift.
func (s *search[S]) unlift(position int) {
	if s.calls[position].span.Op.Known {
		s.relink(s.completion[position])
		s.open++
	}
	s.relink(s.invocation[position])
}

// unlink takes entry out of the list.
func (s *search[S]) unlink(entry int) {
	s.next[s.prev[entry]] = s.next[entry]
	s.prev[s.next[entry]] = s.prev[entry]
}

// relink puts back an entry that unlink took out, between the entries that
// were beside it.
func (s *search[S]) relink(entry int) {
	s.next[s.prev[entry]] = entry
	s.prev[s.next[entry]] = entry
}

// fault returns the fault of the model's function that the search called
// when it panicked with v, or ended the goroutine for a nil v. The fault
// names the call that the search stepped.
func (s *search[S]) fault(v any) error {
	if s.stepped < 0 {
		if v == nil {
			return fault.In(linearizableOp, fault.Of(ErrModel, "the model's %s ends its goroutine", s.calling))
		}
		return fault.In(linearizableOp, fault.Of(ErrModel, "the model's %s panics with %v", s.calling, v))
	}
	c := s.calls[s.stepped].span
	err := fault.Of(ErrModel, "the model's %s ends its goroutine on %q", s.calling, c.Op.Operation)
	if v != nil {
		err = fault.Of(ErrModel, "the model's %s panics on %q with %v", s.calling, c.Op.Operation, v)
	}
	return fault.In(linearizableOp, fault.At(err, fault.Field(callsField), fault.Index(c.Call)))
}
