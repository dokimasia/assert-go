// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"slices"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// The odds with which a machine's swarm keeps an action, before the draw
// conditions the choices on keeping one: 1 in 2, the coin of the swarm
// testing paper, which the definition fixes.
const (
	keepNum = 1
	keepDen = 2
)

// keptBounds are the bounds [1, 1] of the swarm choice of the last action
// while no earlier action is kept.
var keptBounds = choice.MustIntegerBounds(choice.UintOf(1), choice.UintOf(1))

// MachineStep is one step of a machine: the action that it took, the client
// that ran it in a concurrent section, and whether it is a step of the
// drain.
type MachineStep struct {
	// Action is the name of the step's action.
	Action string
	// Client is the client of a step of a concurrent section, and -1 for any
	// other step.
	Client int
	// Drain reports a step of the drain.
	Drain bool
}

// RecordedStep is a step that a case recorded, with the number of draws that
// the case recorded before it.
type RecordedStep struct {
	MachineStep
	// Draws is the number of draws before the step.
	Draws int
}

// Keep returns the swarm's choice whether the case keeps an action, which
// decides structure: true keeps it. kept states whether the case kept an
// earlier action, and remaining counts this action and the actions after
// it. The choice has the bounds [1, 1] for the last action while no earlier
// one is kept, and [0, 1] otherwise. Its edge is 1, so the edge phase keeps
// every action, and its target 0. The random phase draws it as random.Keep
// does with odds of 1 in 2. It ends the calling goroutine as [Case.Integer]
// does.
func (c *Case) Keep(kept bool, remaining int) bool {
	b := bitBounds
	if !kept && remaining == 1 {
		b = keptBounds
	}
	r := request{
		bounds: choice.OfInteger(b), structure: true, edge: choice.UintOf(1), drawing: byKeep, num: keepNum,
		den: keepDen, kept: kept, remaining: remaining,
	}
	return c.choose(r).Integer == choice.UintOf(1)
}

// Weighted returns the case's choice of an index into weights, which
// decides structure, with edge 0. The random phase draws it as
// random.Weighted does, so a list of one weight consumes nothing. weights
// is not empty, and each weight is 1 or more. It ends the calling goroutine
// as [Case.Integer] does.
func (c *Case) Weighted(weights []uint64) int {
	b := choice.MustIntegerBounds(choice.Int{}, choice.UintOf(uint64(len(weights)-1)))
	r := request{bounds: choice.OfInteger(b), structure: true, drawing: byWeighted, weights: weights}
	return int(c.choose(r).Integer.Magnitude())
}

// Continue returns the decision whether a machine's run of count steps
// takes another, by the collection rule with the minimum 0, the maximum
// maxSteps, and the average mean: a choice that decides structure, whose
// edge gives the run one step. maxSteps is 0 or more, and mean is too. It
// ends the calling goroutine as [Case.Integer] does.
func (c *Case) Continue(count, maxSteps, mean int) bool {
	sizes, _ := choice.NewSizes(0, maxSteps)
	return c.more(sizes, count, mean)
}

// Uniform returns a choice below n that decides structure, such as the
// client of a step, with edge edge. The random phase draws it uniformly, so
// a choice below 1 consumes nothing. n is 1 or more. It ends the calling
// goroutine as [Case.Integer] does.
func (c *Case) Uniform(n, edge uint64) uint64 {
	b := choice.MustIntegerBounds(choice.Int{}, choice.UintOf(n-1))
	r := request{bounds: choice.OfInteger(b), structure: true, edge: choice.UintOf(edge), drawing: byUniform}
	return c.choose(r).Integer.Magnitude()
}

// SpanFrom calls f inside a span labelled label that starts at the choice at
// start, which [Case.Position] returned before the case made it, and ends
// after the last choice f makes. A machine's step is such a span, from its
// continue flag to the end of its run. The span closes when f returns or
// ends the goroutine.
func (c *Case) SpanFrom(start int, label string, f func()) {
	span := c.openSpanAt(label, start)
	defer c.closeSpan(span)
	f()
}

// Step records a step of a machine, after the draws that the case recorded
// so far.
func (c *Case) Step(s MachineStep) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.taken = append(c.taken, RecordedStep{MachineStep: s, Draws: len(c.draws)})
}

// Steps returns a copy of the steps that the case recorded, in order.
func (c *Case) Steps() []RecordedStep {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.taken)
}

// Repeat asks the run to run the case up to n times, as a machine whose
// concurrent section runs on real threads does. When the case passes, the
// run runs the body again on the case's choices, outside the case tree,
// until a run fails or n runs in all have passed, and the case is the first
// run that failed. Every run of a case asks again, so the replay that
// confirms a failure and every shrink candidate repeat the same way. The
// case keeps the largest n that it asked for.
func (c *Case) Repeat(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.repeat = max(c.repeat, n)
}

// repeats returns the number of runs that the case asked for, 1 for none.
func (c *Case) repeats() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return max(c.repeat, 1)
}
