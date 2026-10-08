// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package random

import (
	"cmp"
	"math"
	"slices"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// The branches and the sizes of a float draw.
const (
	// floatBranches is the number of branches that a float draw picks
	// from with Below.
	floatBranches = 8
	// edgeBranch is the branch that takes an edge value.
	edgeBranch = 0
	// lastIntegralBranch is the last of the branches 1 to 3, which draw an
	// integral value.
	lastIntegralBranch = 3
	// signValues is the number of values of a sign bit.
	signValues = 2
	// maxFloatEdges is the number of candidate edge values of float
	// bounds, NaN included.
	maxFloatEdges = 6
)

// Float draws a float inside b.
//
// Below(8) picks the branch. Branch 0 takes an edge value, chosen with
// Below(the number of edges). Branches 1 to 3 draw an integer as [Integer]
// draws, over the integers that [choice.FloatBounds.Integers] returns.
// The other branches, and an integral branch for bounds without such an
// integer, assemble the value from its bits.
//
// The assembly draws a sign with Below(2), a biased exponent with
// Below(2^ExponentBits) and a mantissa with Below(2^MantissaBits), in that
// order. A NaN becomes [choice.NaN] when b admits NaN and the target
// otherwise. A value below the lower bound becomes the lower bound, and a
// value above the upper bound becomes the upper bound, compared as
// numbers, so -0 stays -0 inside bounds from +0.
func Float(s *Source, b choice.FloatBounds) float64 {
	branch := s.Below(floatBranches)
	if branch == edgeBranch {
		var edges [maxFloatEdges]float64
		values := AppendFloatEdges(edges[:0], b)
		return values[s.Below(uint64(len(values)))]
	}
	if branch <= lastIntegralBranch {
		if integers, ok := b.Integers(); ok {
			return Integer(s, integers).Float64()
		}
	}
	return fromParts(s, b)
}

// AppendFloatEdges appends the edge values of b to dst and returns the
// extended slice. The edge values are the target, the lower bound, the
// upper bound, the next value of the width above the target and the next
// below it, in that order, each once, and only those that b admits. NaN
// follows them when b admits it.
func AppendFloatEdges(dst []float64, b choice.FloatBounds) []float64 {
	target, w := b.Target(), b.Width()
	candidates := [...]float64{target, b.Lo(), b.Hi(), choice.NextUp(target, w), choice.NextDown(target, w)}
	start := len(dst)
	for _, v := range candidates {
		if b.Admits(v) && !containsFloat(dst[start:], v) {
			dst = append(dst, v)
		}
	}
	if b.NaNPolicy() == choice.AdmitNaN {
		dst = append(dst, choice.NaN())
	}
	return dst
}

// containsFloat reports whether values contains a value that
// [choice.SameFloat] equates with v.
func containsFloat(values []float64, v float64) bool {
	return slices.ContainsFunc(values, func(e float64) bool { return choice.SameFloat(e, v) })
}

// fromParts assembles a value of b's width from a sign, a biased exponent
// and a mantissa, and fits it to b, as [Float] states.
func fromParts(s *Source, b choice.FloatBounds) float64 {
	w := b.Width()
	sign := s.Below(signValues)
	exponent := s.Below(1 << w.ExponentBits())
	mantissa := s.Below(1 << w.MantissaBits())
	value := w.FromParts(sign, exponent, mantissa)
	if math.IsNaN(value) {
		if b.NaNPolicy() == choice.AdmitNaN {
			return choice.NaN()
		}
		return b.Target()
	}
	if b.Admits(value) {
		return value
	}
	if cmp.Less(value, b.Lo()) {
		return b.Lo()
	}
	return b.Hi()
}
