// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"math"
	"slices"
	"time"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/coverage"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/record"
)

// The odds of a campaign's decisions. The definition leaves them to each
// library.
const (
	// mutateNum and mutateDen are the odds that a case of a campaign with a
	// pool mutates a member of the pool: 3 in 4.
	mutateNum, mutateDen = 3, 4
	// mutationKinds is the number of kinds of mutation, among which a mutated
	// case chooses uniformly.
	mutationKinds = 4
)

// The kinds of mutation of a pool member.
const (
	// redraw draws one choice again from its bounds.
	redraw = 0
	// deleteSpan deletes the choices of one span.
	deleteSpan = 1
	// repeatSpan repeats the choices of one span after it.
	repeatSpan = 2
	// replaceSpan replaces the choices of one span with those of a span of
	// the same label of a member.
	replaceSpan = 3
)

// campaignStream is the index of the stream of a campaign's own decisions
// among the streams of its seed: the last index, above the index of every
// random case that a campaign runs.
const campaignStream = math.MaxUint64

// Campaign runs a campaign of body under s: the case of s.Draws, the
// examples and the stored cases, in that order, and then cases until
// s.Budget has passed on s.BudgetClock, one at a time.
//
// A valid case joins the campaign's pool when it counts a label that no
// earlier case counted, records a fingerprint that no earlier case
// recorded, or records a score above the best of its label. Three cases in
// four of a campaign with a pool mutate a member of the pool, chosen
// uniformly, and the fourth is the next random case of the seed: random
// case i for i = 0, 1, 2 and on, past s.Cases. A mutation draws one choice
// of the member again from its bounds, deletes a span, repeats a span, or
// replaces a span with a span of the same label of a member, each with odds
// of 1 in 4. A mutated case replays its choices, and each request past them
// draws from the campaign's own stream.
//
// A failing case whose identity no earlier failure had is replayed, shrunk
// and explained as [Run] concludes its failure, s.Concluded receives the
// result, and the campaign goes on. A replay that differs ends the campaign
// as flaky. A refused entry of s.Draws ends it with the refusal.
//
// When the budget has passed, the campaign returns a counterexample of the
// first failure that it concluded, with every later one among its others,
// or the outcome that the last check of a run decides over every valid case
// of the campaign. The result counts the campaign's cases, states s.Seed,
// and states the runs of the stored cases.
//
// The slot of s takes the calls of every case, those of the random and the
// mutated cases under the phase random.
func Campaign(body Body, s Settings) Result {
	s = withClocks(s)
	c := &campaign{
		body:         body,
		s:            s,
		t:            newTally(s),
		end:          s.BudgetClock.Now().Add(s.Budget),
		source:       random.ForCase(s.Seed, campaignStream),
		labels:       make(map[string]struct{}),
		fingerprints: make(map[uint64]struct{}),
		best:         make(map[string]float64),
		identities:   make(map[Identity]struct{}),
	}
	if r, ended := c.known(); ended {
		return r
	}
	for s.BudgetClock.Now().Before(c.end) {
		if r, ended := c.take(c.next(), record.Random); ended {
			return r
		}
	}
	return c.result()
}

// campaign is the state of one campaign.
type campaign struct {
	// body is the property's body, and s the settings of the campaign.
	body Body
	s    Settings
	// t counts the cases.
	t *tally
	// end is when the budget passes, on the budget's clock.
	end time.Time
	// source is the campaign's own stream, of its decisions and of the
	// requests past a mutated case's choices.
	source random.Source
	// index is the index of the next random case of the seed.
	index uint64
	// pool are the members of the pool, in the order they joined.
	pool []member
	// labels and fingerprints are those that a valid case counted or
	// recorded, and best is the best score of each label.
	labels       map[string]struct{}
	fingerprints map[uint64]struct{}
	best         map[string]float64
	// identities are the identities of the failures concluded so far.
	identities map[Identity]struct{}
	// found is the result of the first failure concluded, with every later
	// one among its others, and nil before the first.
	found *Result
	// stored are the runs of the stored cases.
	stored []Execution
}

// member is a case of the pool: its choices with the bounds of each, and
// its spans.
type member struct {
	// choices are the choices of the case.
	choices []choice.Choice
	// bounds are the bounds of the request of each choice.
	bounds []choice.Bounds
	// spans are the spans of the case.
	spans []Span
}

// known runs the case of the Draws entries, the examples and the stored
// cases, and returns the result that ends the campaign.
func (c *campaign) known() (Result, bool) {
	if len(c.s.Draws) > 0 {
		if r, ended := c.take(executeDraws(c.body, c.s), record.Example); ended {
			return r, true
		}
	}
	for _, ex := range c.s.Examples {
		if r, ended := c.take(executeExample(c.body, c.s, ex), record.Example); ended {
			return r, true
		}
	}
	for _, choices := range c.s.Stored {
		e := execute(c.body, replaying{choices: choices}, c.s)
		c.stored = append(c.stored, e)
		if r, ended := c.take(e, record.Stored); ended {
			return r, true
		}
	}
	return Result{}, false
}

// next runs the next case of the exploration: a mutation of a member of the
// pool, or the next random case of the seed.
func (c *campaign) next() Execution {
	if len(c.pool) > 0 && c.source.Coin(mutateNum, mutateDen) {
		return execute(c.body, c.mutated(), c.s)
	}
	e := execute(c.body, newGenerating(random.ForCase(c.s.Seed, c.index)), c.s)
	c.index++
	return e
}

// mutated returns the provider of a mutation of a member of the pool,
// chosen uniformly. A member without spans takes the mutation that draws a
// choice again, and so does a span whose member of the replacement has no
// span of its label. A member without choices takes no mutation, and every
// request of its case draws from the campaign's stream.
func (c *campaign) mutated() provider {
	m := c.pool[c.source.Below(uint64(len(c.pool)))]
	choices := slices.Clone(m.choices)
	if len(choices) == 0 {
		return &mutating{source: &c.source}
	}
	kind := c.source.Below(mutationKinds)
	if len(m.spans) == 0 {
		kind = redraw
	}
	var span Span
	if kind != redraw {
		span = m.spans[c.source.Below(uint64(len(m.spans)))]
	}
	if kind == replaceSpan {
		donor := c.pool[c.source.Below(uint64(len(c.pool)))]
		same := slices.DeleteFunc(slices.Clone(donor.spans), func(sp Span) bool { return sp.Label != span.Label })
		if len(same) == 0 {
			kind = redraw
		} else {
			d := same[c.source.Below(uint64(len(same)))]
			choices = slices.Concat(choices[:span.Start], donor.choices[d.Start:d.End], choices[span.End:])
		}
	}
	if kind == redraw {
		i := c.source.Below(uint64(len(choices)))
		r := request{bounds: m.bounds[i]}
		choices[i] = r.fromBounds(&c.source)
	}
	if kind == deleteSpan {
		choices = slices.Delete(choices, span.Start, span.End)
	}
	if kind == repeatSpan {
		choices = slices.Insert(choices, span.End, m.choices[span.Start:span.End]...)
	}
	return &mutating{choices: choices, source: &c.source}
}

// take counts the case e, whose calls the slot of the campaign takes under
// phase. It concludes a failure, admits a valid case to the pool, and
// returns the result that ends the campaign: the refusal of an entry of
// Draws, or a failure whose replay differs.
func (c *campaign) take(e Execution, phase record.Phase) (Result, bool) {
	r, ended := c.t.take(e, phase)
	if e.Status == CaseFailed {
		return c.fail(e)
	}
	if ended {
		return r, true
	}
	if e.Status == CasePassed {
		c.admit(e)
	}
	return Result{}, false
}

// fail concludes the failing case e when no earlier failure had its
// identity, and returns the flaky result that a replay that differs ends
// the campaign with. The other failures that the shrink of e finds are
// among its others, except those of an identity concluded before.
func (c *campaign) fail(e Execution) (Result, bool) {
	if _, concluded := c.identities[e.Identity]; concluded {
		return Result{}, false
	}
	r := conclude(c.body, c.s, Result{Outcome: Counterexample, Seed: c.s.Seed, Failing: &e})
	if r.Outcome == Flaky {
		flaky := c.t.result(Flaky)
		flaky.Failing, flaky.Divergence, flaky.Stored = r.Failing, r.Divergence, c.stored
		return flaky, true
	}
	c.identities[e.Identity] = struct{}{}
	others := r.Others[:0]
	for _, o := range r.Others {
		if _, concluded := c.identities[o.Identity]; !concluded {
			c.identities[o.Identity] = struct{}{}
			others = append(others, o)
		}
	}
	r.Others = others
	if c.s.Concluded != nil {
		c.s.Concluded(r)
	}
	if c.found == nil {
		c.found = &r
		return Result{}, false
	}
	c.found.Others = append(append(c.found.Others, *r.Failing), r.Others...)
	return Result{}, false
}

// admit adds the valid case e to the pool when it counts a label, records a
// fingerprint or scores above the best of a label for the first time.
func (c *campaign) admit(e Execution) {
	novel := false
	for _, label := range e.Case.Labels() {
		if _, counted := c.labels[label]; !counted {
			c.labels[label], novel = struct{}{}, true
		}
	}
	for _, f := range e.Case.Fingerprints() {
		if _, recorded := c.fingerprints[f]; !recorded {
			c.fingerprints[f], novel = struct{}{}, true
		}
	}
	for label, score := range e.Case.Targets() {
		if best, scored := c.best[label]; !scored || score > best {
			c.best[label], novel = score, true
		}
	}
	if !novel {
		return
	}
	e.Case.mu.Lock()
	defer e.Case.mu.Unlock()
	bounds := make([]choice.Bounds, len(e.Case.requests))
	for i, r := range e.Case.requests {
		bounds[i] = r.bounds
	}
	c.pool = append(c.pool, member{
		choices: slices.Clone(e.Case.choices), bounds: bounds, spans: slices.Clone(e.Case.spans),
	})
}

// result returns the result of a campaign whose budget has passed.
func (c *campaign) result() Result {
	if c.found != nil {
		r := *c.found
		r.Cases, r.Rejected, r.Stored = c.t.valid, c.t.rejected, c.stored
		return r
	}
	r, ended := c.t.missing()
	if !ended {
		r = c.t.result(Passed)
		if shortfall, _ := shortfallOf(c.t, c.s.Requirements, coverage.Final); shortfall != nil {
			r.Outcome, r.Shortfall = CoverageUnmet, shortfall
		}
	}
	r.Stored = c.stored
	return r
}

// mutating is the provider of a mutated case of a campaign. It replays the
// case's choices, each fitted to its request's bounds, and draws each
// request past them from the campaign's stream.
type mutating struct {
	// choices are the mutated choices.
	choices []choice.Choice
	// source is the campaign's stream.
	source *random.Source
}

// value returns the mutated choice at index, fitted to r's bounds, or r's
// draw from the campaign's stream past the last one.
func (m *mutating) value(r request, index int) choice.Choice {
	if index < len(m.choices) {
		return r.bounds.Coerce(m.choices[index])
	}
	return r.draw(m.source)
}
