// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import "go.dokimi.dev/assert/internal/prop/choice"

// boundary is the boundary that every value choice of one edge case
// takes.
type boundary uint8

const (
	// atMin gives an integer or a float its lower bound, and a sequence
	// elements of 0.
	atMin boundary = 0
	// atMax gives an integer or a float its upper bound, and a sequence
	// elements of k - 1.
	atMax boundary = 1
	// aboveTarget gives an integer the value above its target, a float the
	// next value of its width above the target, and a sequence elements of
	// 1.
	aboveTarget boundary = 2
	// belowTarget gives an integer the value below its target, a float the
	// next value of its width below the target, and a sequence elements of
	// its target, 0.
	belowTarget boundary = 3
)

// boundaries are the edge cases of a run, in the order it tries them.
var boundaries = [...]boundary{atMin, atMax, aboveTarget, belowTarget}

// edge is a provider that gives every choice its value at one boundary.
//
// A choice that decides structure takes the edge its request states, so a
// collection gets one element, a one-of its first alternative and an
// optional its value. A value choice takes the boundary of its case, and
// the target when its bounds do not admit that value. A sequence has one
// element, or as many as its minimum length when that is more, and none
// when its maximum length is 0.
type edge struct {
	// at is the boundary of the case.
	at boundary
}

// value returns r's edge, or its value at the boundary.
func (e edge) value(r request, _ int) choice.Choice {
	if r.structure {
		v := choice.Choice{Kind: choice.Integer, Integer: r.edge}
		if r.bounds.Admits(v) {
			return v
		}
		return r.bounds.Target()
	}
	if r.bounds.Kind() == choice.Integer {
		return choice.Choice{Kind: choice.Integer, Integer: e.integer(r.bounds.Integer())}
	}
	if r.bounds.Kind() == choice.Float {
		return choice.Choice{Kind: choice.Float, Float: e.float(r.bounds.Float())}
	}
	return choice.Choice{Kind: choice.Sequence, Sequence: e.sequence(r.bounds.Sequence())}
}

// integer returns an integer choice's value at the boundary.
func (e edge) integer(b choice.IntegerBounds) choice.Int {
	if e.at == atMin {
		return b.Lo()
	}
	if e.at == atMax {
		return b.Hi()
	}
	if e.at == aboveTarget && b.Above() > 0 {
		return b.Target().Add(1)
	}
	if e.at == belowTarget && b.Below() > 0 {
		return b.Target().Sub(1)
	}
	return b.Target()
}

// float returns a float choice's value at the boundary.
func (e edge) float(b choice.FloatBounds) float64 {
	v := b.Hi()
	if e.at == atMin {
		v = b.Lo()
	}
	if e.at == aboveTarget {
		v = choice.NextUp(b.Target(), b.Width())
	}
	if e.at == belowTarget {
		v = choice.NextDown(b.Target(), b.Width())
	}
	if b.Admits(v) {
		return v
	}
	return b.Target()
}

// sequence returns a sequence choice's value at the boundary.
func (e edge) sequence(b choice.SequenceBounds) []uint32 {
	element := uint32(0)
	if e.at == atMax {
		element = b.K() - 1
	}
	if e.at == aboveTarget && b.K() > 1 {
		element = 1
	}
	elements := make([]uint32, b.Sizes().Clamp(1))
	for i := range elements {
		elements[i] = element
	}
	return elements
}
