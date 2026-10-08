// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"cmp"
	"iter"
	"slices"
	"sync"
	"time"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/internal/record"
)

// DefaultShrink is the number of runs that shrinking every failure of a
// run may spend when the caller states no number.
const DefaultShrink = 2000

// node is one recorded choice, with what its case recorded of the request
// for it.
type node struct {
	// r is what the case recorded of the request.
	r recorded
	// c is the choice.
	c choice.Choice
}

// key returns the node's sort key under its request's bounds.
func (n node) key() choice.Key {
	return n.r.bounds.Key(n.c)
}

// candidate is a case smaller than the best one, with choices new to the
// shrinker: its choices with their requests, and its token.
type candidate struct {
	// nodes are the choices with their requests.
	nodes []node
	// choices are the choices alone.
	choices []choice.Choice
	// token is the token of the choices.
	token string
}

// failure is the smallest case found so far that fails with one identity.
type failure struct {
	// identity is the failure's identity.
	identity Identity
	// execution is the run of the smallest case.
	execution Execution
	// nodes are the case's choices with their requests.
	nodes []node
	// spans are the case's spans, which the passes read and do not change.
	spans []Span
}

// shrinker shrinks every failure of one run over one budget.
//
// A case is smaller than another when its choice sequence is shorter, or
// equally long and smaller in the first choice where the two differ.
// Choices of one kind compare by their sort keys, and choices of different
// kinds by kind: integer, then float, then sequence.
//
// The shrinker runs rounds of its passes over the best case of one failure,
// in the order shrink lists them, while each round accepts a candidate.
// deleteAndLower runs only in a round in which no other pass accepted a
// candidate. A candidate runs only when it is smaller than the best case
// and no earlier candidate had the same choices. It is accepted when its
// run fails with the same identity and records a smaller case. A run that
// fails with another identity is kept as a further failure. The failures
// are shrunk one at a time, the one with the smallest case first, and they
// share one budget of runs.
//
// A pass that lists its candidates runs up to Workers of them at once and
// takes their runs in the pass's order, as one worker would have run them.
// It stops at the first accepted one, and charges the budget only for the
// runs up to it.
//
// Each run takes a spare case, whose storage the run starts over. A run
// whose failure the shrinker discards returns its case to the spares once
// the shrinker has read it, so the cases of a shrink grow their storage
// once.
type shrinker struct {
	// body is the property's body.
	body Body
	// s are the run's settings.
	s Settings
	// workers is the number of candidates that run at once, 1 or more.
	workers int
	// deadline is when shrinking stops on the shrink clock, and the zero
	// time for no limit.
	deadline time.Time
	// limit is the number of runs that the budget allows, which the run
	// count becomes once the time is spent.
	limit int
	// runs counts the runs spent.
	runs int
	// failures are the smallest case of each identity found.
	failures map[Identity]*failure
	// found are the identities in the order they were found.
	found []Identity
	// sizes are the token of every candidate run so far, with the number
	// of choices its run recorded.
	sizes map[string]int
	// target is the identity being shrunk.
	target Identity
	// spare are the cases of runs that the shrinker no longer reads, for
	// later runs to start over.
	spare []*Case
	// batch is the storage of the choice sequences of one batch of runs.
	batch [][]choice.Choice
	// executions is the storage of the runs that runAll returns, valid
	// until its next call.
	executions []Execution
	// generating supplies the values of each explanation filling.
	generating *generating
}

// newShrinker returns a shrinker that starts from the first failing case.
func newShrinker(body Body, first Execution, s Settings) *shrinker {
	sh := &shrinker{
		body:       body,
		s:          s,
		workers:    max(s.Workers, 1),
		limit:      s.Shrink,
		failures:   make(map[Identity]*failure),
		sizes:      map[string]int{token.Encode(first.Case.Choices()): len(first.Case.Choices())},
		target:     first.Identity,
		generating: newGenerating(random.Source{}),
	}
	if s.ShrinkTime > 0 {
		sh.deadline = s.ShrinkClock.Now().Add(s.ShrinkTime)
	}
	sh.record(first)
	return sh
}

// best returns the failure being shrunk.
func (sh *shrinker) best() *failure {
	return sh.failures[sh.target]
}

// nodes returns the best case's choices.
func (sh *shrinker) nodes() []node {
	return sh.best().nodes
}

// spans returns the best case's spans, which the caller does not change.
func (sh *shrinker) spans() []Span {
	return sh.best().spans
}

// record keeps a failing run when it is its identity's first or smallest,
// and reports whether it kept the run.
func (sh *shrinker) record(e Execution) bool {
	nodes := nodesOf(e.Case)
	known, ok := sh.failures[e.Identity]
	if !ok {
		sh.found = append(sh.found, e.Identity)
	}
	if ok && compareNodes(nodes, known.nodes) != -1 {
		return false
	}
	sh.failures[e.Identity] = &failure{identity: e.Identity, execution: e, nodes: nodes, spans: e.Case.Spans()}
	return true
}

// room returns the runs left in the budget. Once the time is spent it
// returns 0, and from then on.
func (sh *shrinker) room() int {
	if !sh.deadline.IsZero() && sh.s.ShrinkClock.Now().After(sh.deadline) {
		sh.deadline, sh.limit = time.Time{}, sh.runs
	}
	return sh.limit - sh.runs
}

// spent reports whether the budget or the time is spent. From then on the
// shrinker runs nothing, no pass starts, and a pass that walks the choices
// stops at the next one.
func (sh *shrinker) spent() bool {
	return sh.room() == 0
}

// run runs the body on choices, with every repeat that the case asks for
// within the budget, and charges the budget, as charge does. It reports
// false, and runs nothing, once the budget or the time is spent. The caller
// releases the run's case once it no longer reads it.
func (sh *shrinker) run(choices []choice.Choice, phase record.Phase) (Execution, bool) {
	room := sh.room()
	if room == 0 {
		return Execution{}, false
	}
	return sh.charge(runOnce(sh.replaying(choices), sh.body), room, phase), true
}

// charge runs the repeats that the case of e, which ran once, asks for, up
// to room runs in all, charges the budget one run for each run of the body,
// and hands the calls of the runs to the slot of the run under phase. A case
// that asks for more runs than room allows ends as its runs within room
// ended, as one worker that ran it with that room left would end it.
func (sh *shrinker) charge(e Execution, room int, phase record.Phase) Execution {
	e = repeat(e, sh.body, room)
	sh.runs += e.runs()
	e.take(sh.s.Slot, phase)
	return e
}

// runAll runs the body once on each of one or more choice sequences at
// once, the first on the calling goroutine, and returns their runs in
// order, valid until its next call. The caller charges the budget for each
// run it takes, which runs the repeats that its case asks for, and releases
// the case of each run whose failure it discards.
func (sh *shrinker) runAll(choices [][]choice.Choice) []Execution {
	runs := slices.Grow(sh.executions[:0], len(choices))[:len(choices)]
	for i, c := range choices {
		runs[i] = Execution{Case: sh.replaying(c)}
	}
	var wg sync.WaitGroup
	for i := range runs[1:] {
		wg.Go(func() { runs[i+1] = runOnce(runs[i+1].Case, sh.body) })
	}
	runs[0] = runOnce(runs[0].Case, sh.body)
	wg.Wait()
	sh.executions = runs
	return runs
}

// replaying returns a spare case, or a new one, started over to replay
// choices outside the case tree.
func (sh *shrinker) replaying(choices []choice.Choice) *Case {
	c := sh.spareCase()
	c.recycleReplaying(choices, sh.s)
	return c
}

// spareCase returns a case that no run of the shrinker reads any more, or
// a new case when none is spare.
func (sh *shrinker) spareCase() *Case {
	last := len(sh.spare) - 1
	if last < 0 {
		return new(Case)
	}
	c := sh.spare[last]
	sh.spare[last] = nil
	sh.spare = sh.spare[:last]
	return c
}

// release makes the cases of runs spare, for later runs to start over. The
// caller reads none of the runs afterwards.
func (sh *shrinker) release(runs ...Execution) {
	for _, e := range runs {
		sh.spare = append(sh.spare, e.Case)
	}
}

// consider runs a candidate that is smaller than the best case and new,
// and reports whether it became the best case.
func (sh *shrinker) consider(nodes []node) bool {
	c, ok := sh.fresh(nodes, nil)
	if !ok {
		return false
	}
	accepted, _ := sh.settle([]candidate{c})
	return accepted
}

// fresh returns nodes as a candidate when they are smaller than the best
// case and neither a run nor a candidate of batch had the same choices.
func (sh *shrinker) fresh(nodes []node, batch []candidate) (candidate, bool) {
	if compareNodes(nodes, sh.nodes()) >= 0 {
		return candidate{}, false
	}
	choices := choicesOf(nodes)
	tok := token.Encode(choices)
	_, tried := sh.sizes[tok]
	if tried || slices.ContainsFunc(batch, func(c candidate) bool { return c.token == tok }) {
		return candidate{}, false
	}
	return candidate{nodes: nodes, choices: choices, token: tok}, true
}

// settle runs the candidates of a non-empty batch that the budget leaves
// room for at once, and takes their runs in order, up to the first run that
// became the best case or the end of the budget. For each run that it
// takes, settle:
//
//   - runs the repeats that its case asks for and charges the budget, as
//     charge does,
//   - records the run's size,
//   - hands the calls of the runs to the slot of the run under the phase
//     shrink,
//   - keeps the run's failure.
//
// It reports whether a run became the best case, and whether the search is
// over: a candidate was accepted, or the budget or the time is spent. The
// cases of the runs whose failures it discards become spare.
func (sh *shrinker) settle(batch []candidate) (accepted, over bool) {
	room := sh.room()
	if room == 0 {
		return false, true
	}
	batch = batch[:min(len(batch), room)]
	sh.batch = sh.batch[:0]
	for _, c := range batch {
		sh.batch = append(sh.batch, c.choices)
	}
	runs := sh.runAll(sh.batch)
	taken := 0
	for ; taken < len(runs) && room > 0; taken++ {
		e := sh.charge(runs[taken], room, record.Shrink)
		room -= e.runs()
		sh.sizes[batch[taken].token] = e.Case.Position()
		if e.Status != CaseFailed {
			sh.release(e)
			continue
		}
		before := sh.best()
		if !sh.record(e) {
			sh.release(e)
		}
		if sh.best() != before {
			sh.release(runs[taken+1:]...)
			return true, true
		}
	}
	sh.release(runs[taken:]...)
	return false, room == 0
}

// first runs candidates in order, up to Workers of them at once, and
// reports whether one became the best case. It stops at that one, or once
// the budget or the time is spent.
func (sh *shrinker) first(candidates iter.Seq[[]node]) bool {
	var batch []candidate
	for nodes := range candidates {
		c, ok := sh.fresh(nodes, batch)
		if !ok {
			continue
		}
		batch = append(batch, c)
		if len(batch) < sh.workers {
			continue
		}
		if accepted, over := sh.settle(batch); over {
			return accepted
		}
		batch = batch[:0]
	}
	if len(batch) > 0 {
		accepted, _ := sh.settle(batch)
		return accepted
	}
	return false
}

// shrinkAll shrinks the failures one at a time, the one with the smallest
// case first, until every failure is done. Once the budget or the time is
// spent, each failure left is done after one round that runs nothing.
func (sh *shrinker) shrinkAll() {
	done := make(map[Identity]bool)
	for {
		var pending []*failure
		for _, identity := range sh.found {
			if !done[identity] {
				pending = append(pending, sh.failures[identity])
			}
		}
		if len(pending) == 0 {
			return
		}
		next := slices.MinFunc(pending, func(a, b *failure) int {
			return cmp.Or(compareNodes(a.nodes, b.nodes), a.identity.Compare(b.identity))
		})
		sh.shrink(next.identity)
		done[next.identity] = true
	}
}

// shrink runs rounds of every pass on one failure until a round accepts
// nothing. deleteAndLower runs only in a round in which no other pass
// accepted a candidate, and a round that it improves is followed by
// another. No pass starts once the budget or the time is spent.
func (sh *shrinker) shrink(identity Identity) {
	sh.target = identity
	passes := []func() bool{
		sh.deleteSpanChunk,
		sh.deleteSpan,
		sh.liftDescendant,
		sh.deleteSpanRun,
		sh.sequenceDelete,
		sh.deleteStructurePair,
		sh.targetSpan,
		sh.minimizeChoice,
		sh.sequenceLower,
		sh.lowerAndDelete,
		sh.sortSiblings,
		sh.redistribute,
		sh.lowerTogether,
		sh.minimizeDuplicates,
		sh.floatSimplify,
	}
	for {
		improved := false
		for _, pass := range passes {
			if sh.spent() {
				return
			}
			if pass() {
				improved = true
			}
		}
		if !improved && !sh.deleteAndLower() {
			return
		}
	}
}

// sweep tries the candidates in order. After an acceptance it starts over
// on the new best case. It reports whether any candidate was accepted.
func (sh *shrinker) sweep(candidates func() iter.Seq[[]node]) bool {
	improved := false
	for sh.first(candidates()) {
		improved = true
	}
	return improved
}

// nodesOf returns a case's choices with their requests.
func nodesOf(c *Case) []node {
	c.mu.Lock()
	defer c.mu.Unlock()
	nodes := make([]node, len(c.choices))
	for i := range c.choices {
		nodes[i] = node{r: c.requests[i], c: c.choices[i]}
	}
	return nodes
}

// choicesOf returns the choices of nodes.
func choicesOf(nodes []node) []choice.Choice {
	choices := make([]choice.Choice, len(nodes))
	for i, n := range nodes {
		choices[i] = n.c
	}
	return choices
}

// compareNodes orders two choice sequences shortlex: the shorter first,
// then by the keys of the first choices that differ. It returns -1, 0 or
// +1.
func compareNodes(a, b []node) int {
	if c := cmp.Compare(len(a), len(b)); c != 0 {
		return c
	}
	for i := range a {
		if c := a[i].key().Compare(b[i].key()); c != 0 {
			return c
		}
	}
	return 0
}

// replaced returns nodes with the choice at index replaced by c.
func replaced(nodes []node, index int, c choice.Choice) []node {
	out := slices.Clone(nodes)
	out[index].c = c
	return out
}

// without returns nodes without the nodes in [start, end).
func without(nodes []node, start, end int) []node {
	return slices.Concat(nodes[:start], nodes[end:])
}

// atTarget returns n with its choice at its target.
func atTarget(n node) node {
	n.c = n.r.bounds.Target()
	return n
}
