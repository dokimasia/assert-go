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
	// byFlag draws the continue flag of a collection or of a machine's run
	// of steps.
	byFlag drawing = 2
	// byKeep draws a machine's swarm choice of an action with the request's
	// odds.
	byKeep drawing = 3
	// byWeighted draws an index by the request's weights.
	byWeighted drawing = 4
	// byUniform draws uniformly from the request's bounds, which start at 0.
	byUniform drawing = 5
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
	// num and den are the odds of a coin and of a swarm choice.
	num, den uint64
	// sizes and count are the sizes of the collection or the run of steps
	// whose continue flag the request is, and the number of its elements or
	// steps so far, and average the length that its flags aim for.
	sizes   choice.Sizes
	count   int
	average int
	// kept and remaining are what a swarm choice depends on: whether an
	// earlier action is kept, and the number of actions from this one to the
	// last.
	kept      bool
	remaining int
	// weights are the weights of an index by weight.
	weights []uint64
}

// recorded is what a case keeps of a request once the choice is made: its
// bounds, and whether the choice decides structure. The shrinker, the
// replay that confirms a failure and the explain phase read nothing else of
// a request.
type recorded struct {
	// bounds are the bounds of the choice.
	bounds choice.Bounds
	// structure reports whether the choice decides structure.
	structure bool
}

// draw returns the value that the random phase draws for r from s.
func (r *request) draw(s *random.Source) choice.Choice {
	switch r.drawing {
	case byBounds:
		return r.fromBounds(s)
	case byCoin:
		return bit(s.Coin(r.num, r.den))
	case byFlag:
		return bit(random.Flag(s, r.sizes, r.count, r.average))
	case byKeep:
		return bit(random.Keep(s, r.num, r.den, r.kept, r.remaining))
	case byWeighted:
		return choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(uint64(random.Weighted(s, r.weights)))}
	}
	return choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(s.UpTo(r.bounds.Integer().Hi().Magnitude()))}
}

// fromBounds returns the value that the random package draws from s in r's
// bounds.
func (r *request) fromBounds(s *random.Source) choice.Choice {
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
