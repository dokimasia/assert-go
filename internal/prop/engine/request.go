// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
)

// drawing is how the random phase draws the value of a request.
type drawing uint8

const (
	// byBounds draws from the request's bounds, as the random package
	// draws an integer, a float or a sequence.
	byBounds drawing = 0
	// byCoin draws 1 with the request's odds and 0 otherwise.
	byCoin drawing = 1
	// byFlag draws the continue flag of a collection.
	byFlag drawing = 2
)

// request is one choice that a generator asks a case for.
type request struct {
	// bounds are the bounds of the choice.
	bounds choice.Bounds
	// edge is the value the edge phase gives a choice that decides
	// structure.
	edge choice.Int
	// structure reports whether the choice decides the structure of what
	// its generator returns, such as a collection's continue flag or a
	// one-of's index, and so has an edge.
	structure bool
	// reuse reports whether the random phase may give the choice an
	// earlier value of its case with the same bounds.
	reuse bool
	// drawing is how the random phase draws the value.
	drawing drawing
	// num and den are the odds of a coin.
	num, den uint64
	// sizes and count are the sizes of the collection whose continue flag
	// the request is, and the number of its elements so far.
	sizes choice.Sizes
	count int
}

// draw returns the value that the random phase draws for r from s.
func (r request) draw(s *random.Source) choice.Choice {
	if r.drawing == byBounds {
		return r.fromBounds(s)
	}
	if r.drawing == byCoin {
		return bit(s.Coin(r.num, r.den))
	}
	return bit(random.Flag(s, r.sizes, r.count))
}

// fromBounds returns the value that the random package draws from s in r's
// bounds.
func (r request) fromBounds(s *random.Source) choice.Choice {
	if r.bounds.Kind() == choice.Integer {
		return choice.Choice{Kind: choice.Integer, Integer: random.Integer(s, r.bounds.Integer())}
	}
	if r.bounds.Kind() == choice.Float {
		return choice.Choice{Kind: choice.Float, Float: random.Float(s, r.bounds.Float())}
	}
	return choice.Choice{Kind: choice.Sequence, Sequence: random.AppendSequence(nil, s, r.bounds.Sequence())}
}

// bit returns the integer choice 1 for true and 0 for false.
func bit(set bool) choice.Choice {
	if set {
		return choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(1)}
	}
	return choice.Choice{Kind: choice.Integer}
}
