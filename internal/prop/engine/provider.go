// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
)

// provider is where the values of a case come from: a random source while
// generating, recorded choices while replaying, a boundary in the edge
// phase, or a fuzzer's bytes.
type provider interface {
	// value returns the value for r at index of the case. The value is a
	// choice that r's bounds admit.
	value(r request, index int) choice.Choice
}

// earlierValue is the value of a reuse request of a case, with the index of
// its choice in the case's record.
type earlierValue struct {
	// index is the index of the choice in the record.
	index int
	// value is the choice.
	value choice.Choice
}

// generating is a provider that draws every value from a random source.
//
// A request marked reuse, whose bounds admit more than one value, may take
// the value of an earlier reuse request of the same case with the same
// bounds, as [random.Reuse] decides. The second of two keys or two
// identifiers then equals the first in at least one case in four. The
// earlier values are those whose choices the record still contains,
// whatever their bounds, so a filter's next attempt never takes a value of
// an attempt it rejected.
//
// A run on one worker serves every random case with one generating
// provider, which reset starts over for each case. The map of earlier
// values keeps its storage across the cases.
type generating struct {
	// source is the stream of the case.
	source random.Source
	// earlier are the values of the case's reuse requests so far, by their
	// bounds, in record order. Only integer requests are marked reuse.
	earlier map[choice.IntegerBounds][]earlierValue
	// next is one past the index of the last earlier value, and 0 for none.
	next int
}

// newGenerating returns a provider that draws from source.
func newGenerating(source random.Source) *generating {
	return &generating{source: source, earlier: make(map[choice.IntegerBounds][]earlierValue)}
}

// reset starts the provider over on source, without earlier values, for
// the next case. It allocates nothing.
func (g *generating) reset(source random.Source) {
	g.source = source
	g.next = 0
	clear(g.earlier)
}

// value returns r's draw from the source, or an earlier value of the case.
// A request at index finds index choices in the record, so the earlier
// values at index and after it, of every bounds, are of choices that a
// rewind removed, and leave the earlier values first.
func (g *generating) value(r request, index int) choice.Choice {
	if index < g.next {
		g.forget(index)
	}
	bounds := r.bounds.Integer()
	if !r.reuse || bounds.Lo() == bounds.Hi() {
		return r.draw(&g.source)
	}
	earlier := g.earlier[bounds]
	var v choice.Choice
	if i, ok := random.Reuse(&g.source, len(earlier)); ok {
		v = earlier[i].value
	} else {
		v = r.draw(&g.source)
	}
	g.earlier[bounds] = append(earlier, earlierValue{index: index, value: v})
	g.next = index + 1
	return v
}

// forget removes the earlier values at index and after it, of every
// bounds.
func (g *generating) forget(index int) {
	for bounds, earlier := range g.earlier {
		for len(earlier) > 0 && earlier[len(earlier)-1].index >= index {
			earlier = earlier[:len(earlier)-1]
		}
		g.earlier[bounds] = earlier
	}
	g.next = index
}

// trailing is a generating provider that keeps the state of its source
// after each value.
//
// A run on more than one worker runs a random case outside the case tree
// and enters it afterwards. A run on one worker stops a repeated case at
// its repeat, and draws the cut of the prefix case from the source as that
// stop left it. The kept states let the run resume the source there.
type trailing struct {
	*generating
	// states are the source before the first value and after each value,
	// in order.
	states []random.Source
}

// newTrailing returns a provider that draws from source and keeps its
// states.
func newTrailing(source random.Source) *trailing {
	return &trailing{generating: newGenerating(source), states: []random.Source{source}}
}

// value returns r's value as generating does, and keeps the state of the
// source after it.
func (t *trailing) value(r request, index int) choice.Choice {
	v := t.generating.value(r, index)
	t.states = append(t.states, t.source)
	return v
}

// after returns the source after its first n values.
func (t *trailing) after(n int) random.Source {
	return t.states[n]
}

// replaying is a provider that reads recorded choices back, in order. A
// recorded choice that does not fit the request is coerced by the
// request's bounds, and a request past the last recorded choice takes the
// target of its bounds.
type replaying struct {
	// choices are the recorded choices.
	choices []choice.Choice
}

// value returns the recorded choice at index, fitted to r's bounds.
func (p replaying) value(r request, index int) choice.Choice {
	if index >= len(p.choices) {
		return r.bounds.Target()
	}
	return r.bounds.Coerce(p.choices[index])
}
