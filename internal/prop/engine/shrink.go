// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"cmp"
	"iter"
	"slices"
	"sync"
	"time"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/token"
)

// DefaultShrink is the number of runs that shrinking every failure of a
// run may spend when the caller states no number.
const DefaultShrink = 2000

// node is one recorded choice, with what its case recorded of the request
// it answered.
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
// The shrinker runs its passes over the best case of one failure, in the
// order shrink lists them, until a whole round accepts no candidate. A
// candidate runs only when it is smaller than the best case and no earlier
// candidate had the same choices. It is accepted when its run fails with
// the same identity and records a smaller case. A run that fails with
// another identity is kept as a further failure. The failures are shrunk
// one at a time, the one with the smallest case first, and they share one
// budget of runs.
//
// A pass that lists its candidates runs up to Workers of them at once and
// takes their runs in the pass's order, as one worker would have run them.
// It stops at the first accepted one, and charges the budget only for the
// runs up to it.
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
}

// newShrinker returns a shrinker that starts from the first failing case.
func newShrinker(body Body, first Execution, s Settings) *shrinker {
	sh := &shrinker{
		body:     body,
		s:        s,
		workers:  max(s.Workers, 1),
		limit:    s.Shrink,
		failures: make(map[Identity]*failure),
		sizes:    map[string]int{token.Encode(first.Case.Choices()): len(first.Case.Choices())},
		target:   first.Identity,
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

// record keeps a failing run when it is its identity's first or smallest.
func (sh *shrinker) record(e Execution) {
	nodes := nodesOf(e.Case)
	known, ok := sh.failures[e.Identity]
	if !ok {
		sh.found = append(sh.found, e.Identity)
	}
	if !ok || compareNodes(nodes, known.nodes) == -1 {
		sh.failures[e.Identity] = &failure{identity: e.Identity, execution: e, nodes: nodes, spans: e.Case.Spans()}
	}
}

// room returns the runs left in the budget. Once the time is spent it
// returns 0, and from then on.
func (sh *shrinker) room() int {
	if !sh.deadline.IsZero() && sh.s.ShrinkClock.Now().After(sh.deadline) {
		sh.deadline, sh.limit = time.Time{}, sh.runs
	}
	return sh.limit - sh.runs
}

// run runs the body on choices, spending one run of the budget. It reports
// false, and runs nothing, once the budget or the time is spent.
func (sh *shrinker) run(choices []choice.Choice) (Execution, bool) {
	if sh.room() == 0 {
		return Execution{}, false
	}
	sh.runs++
	return sh.replay(choices), true
}

// runAll runs the body on each of one or more choice sequences at once,
// the first on the calling goroutine, and returns their runs in order. The
// caller charges the budget for each run it takes.
func (sh *shrinker) runAll(choices [][]choice.Choice) []Execution {
	runs := make([]Execution, len(choices))
	var wg sync.WaitGroup
	for i, c := range choices[1:] {
		wg.Go(func() { runs[i+1] = sh.replay(c) })
	}
	runs[0] = sh.replay(choices[0])
	wg.Wait()
	return runs
}

// replay runs the body on a case that replays choices, outside the case
// tree.
func (sh *shrinker) replay(choices []choice.Choice) Execution {
	return execute(sh.body, replaying{choices: choices}, sh.s.MaxChoices, sh.s.Clock)
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
// room for at once, and takes their runs in order. It charges the budget
// for each run, records each run's size, and keeps each failure. It stops
// at the first run that became the best case. It reports whether one did,
// and whether the search is over: a candidate was accepted, or the budget
// or the time is spent.
func (sh *shrinker) settle(batch []candidate) (accepted, over bool) {
	room := sh.room()
	if room == 0 {
		return false, true
	}
	batch = batch[:min(len(batch), room)]
	choices := make([][]choice.Choice, len(batch))
	for i, c := range batch {
		choices[i] = c.choices
	}
	for i, e := range sh.runAll(choices) {
		sh.runs++
		sh.sizes[batch[i].token] = e.Case.position()
		if e.Status != CaseFailed {
			continue
		}
		before := sh.best()
		sh.record(e)
		if sh.best() != before {
			return true, true
		}
	}
	return false, false
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
// nothing.
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
			if pass() {
				improved = true
			}
		}
		if !improved {
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
