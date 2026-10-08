// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import "go.dokimi.dev/assert/internal/prop/choice"

// Example is a case whose values the caller states, which a run tries after
// the case of [Settings.Draws] and before the stored cases.
type Example struct {
	// Choices are the choices that decode to the case's values, which the
	// case replays.
	Choices []choice.Choice
	// Values are the values of the case's draws, in order, for an input
	// whose generator has no inverse, and nil for an example of choices. Each
	// draw takes the next value and makes no choice. A draw past the last
	// value decodes, and each of its requests takes its target.
	Values []any
}

// valuing is the provider of an example of values. Its draws take the
// values, so a request asks for a choice only once the values run out, and
// it takes its target, as a replay that runs out of choices does.
type valuing struct {
	// values are the values, in draw order.
	values []any
	// taken is the number of values that draws have taken.
	taken int
}

// value returns the target of r, which no stated value serves.
func (*valuing) value(r request, _ int) choice.Choice {
	return r.bounds.Target()
}

// executeExample calls body once on the case of ex, outside the case tree,
// and returns how the case ended.
func executeExample(body Body, s Settings, ex Example) Execution {
	if ex.Values == nil {
		return execute(body, replaying{choices: ex.Choices}, s)
	}
	p := &valuing{values: ex.Values}
	c := newCase(p, s, nil)
	c.valuing = p
	return finish(c, body)
}

// stated returns the next value of c, an example of values, which a draw
// takes, and false once the values ran out.
func (c *Case) stated() (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.valuing
	if p.taken == len(p.values) {
		return nil, false
	}
	p.taken++
	return p.values[p.taken-1], true
}

// Valued reports whether the case is an example of values, whose draws took
// stated values without a choice. Such a case has no replay token and no
// store entry, and a run reports its failure as found.
func (c *Case) Valued() bool {
	return c.valuing != nil
}
