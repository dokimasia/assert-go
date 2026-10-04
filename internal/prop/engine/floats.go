// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"math"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// maxIntegral is 2^53 - 1, the largest magnitude at which the float pass
// moves an integral float as an integer.
const maxIntegral = 9007199254740991

// floatSimplify simplifies each float, in order: fewer fractional bits,
// then a smaller integer. A fraction takes the first rounding that is
// accepted. An integral value below 2^53 in magnitude, whose target is
// integral too, then moves towards its target as an integer does.
func (sh *shrinker) floatSimplify() bool {
	improved := false
	for index := 0; index < len(sh.nodes()) && !sh.spent(); index++ {
		nodes := sh.nodes()
		n := nodes[index]
		if n.r.bounds.Kind() != choice.Float {
			continue
		}
		rounded := func(yield func([]node) bool) {
			for _, value := range roundings(n.c.Float, n.r.bounds.Float()) {
				if !yield(replaced(nodes, index, floatChoice(value))) {
					return
				}
			}
		}
		improved = sh.first(rounded) || improved
		improved = sh.lowerFloat(index) || improved
	}
	return improved
}

// roundings returns a fraction rounded to fewer fractional bits, from 0
// bits up. At each number of bits it returns the value rounded towards the
// target, then away from it, where the bounds admit them. An integral
// value, an infinity and NaN have none.
func roundings(value float64, b choice.FloatBounds) []float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	var out []float64
	// The value is above its target when their difference is negative. A
	// value at its target has no rounding smaller than it, so the order of
	// its roundings does not matter.
	downward := math.Signbit(b.Target() - value)
	for fraction := range int(choice.FractionBits(value)) {
		scaled := math.Ldexp(value, fraction)
		first, second := math.Ceil(scaled), math.Floor(scaled)
		if downward {
			first, second = second, first
		}
		for _, rounded := range [...]float64{first, second} {
			if candidate := math.Ldexp(rounded, -fraction); b.Admits(candidate) {
				out = append(out, candidate)
			}
		}
	}
	return out
}

// lowerFloat moves an integral float at index towards its target, as an
// integer. The search runs over the integers of the bounds up to 2^53 - 1
// in magnitude, in their key order, and skips a value the float's width
// cannot state. The bounds then contain an integer of that magnitude, so
// their target is the integer of least magnitude among them.
func (sh *shrinker) lowerFloat(index int) bool {
	nodes := sh.nodes()
	b, value := nodes[index].r.bounds.Float(), nodes[index].c.Float
	if value != math.Trunc(value) || math.Abs(value) > maxIntegral {
		return false
	}
	lo, hi := max(math.Ceil(b.Lo()), -maxIntegral), min(math.Floor(b.Hi()), maxIntegral)
	integers := choice.MustIntegerBounds(choice.IntOf(int64(lo)), choice.IntOf(int64(hi)))
	at := func(rank uint64) bool {
		candidate := integers.AtRank(rank).Float64()
		if !b.Admits(candidate) {
			return false
		}
		return sh.consider(replaced(nodes, index, floatChoice(candidate)))
	}
	return search(integers.Rank(choice.IntOf(int64(value))), at)
}

// floatChoice returns the float choice of v.
func floatChoice(v float64) choice.Choice {
	return choice.Choice{Kind: choice.Float, Float: v}
}
