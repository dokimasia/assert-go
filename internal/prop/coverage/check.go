// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package coverage

import (
	"fmt"
	"math"
)

// The constants of the test.
const (
	// Z is the standard normal quantile at 1 - 5e-10: QuickCheck's
	// certainty of 10^9, split between the two tails.
	Z = 6.109410191663286
	// Tolerance is the fraction of the required share that the lower bound
	// must reach for a requirement to be met.
	Tolerance = 0.9
)

// checks are the multiples of the cases setting after which the runner
// checks every requirement, in order.
var checks = [...]int{1, 2, 4, 8}

// Checks returns the multiples of the cases setting after which the
// runner checks every requirement: 1, 2, 4 and 8. Each doubles the one
// before it, and the last is the last check of a run.
func Checks() [4]int {
	return checks
}

// Bound returns the Wilson score bound of k successes in n trials at
// quantile z: the upper bound for a positive z and the lower bound for a
// negative one. It panics when n is not positive or k is outside [0, n].
//
// Bound evaluates the definition's expression in its written order, and
// rounds each product before an addition, so every platform returns the
// same bits.
func Bound(k, n int, z float64) float64 {
	checkShare(k, n)
	p := float64(k) / float64(n)
	a := z * z / float64(n)
	centre := p + a/2
	spread := float64(z * math.Sqrt(p*(1-p)/float64(n)+a/(4*float64(n))))
	return (centre + spread) / (1 + a)
}

// Decide returns the verdict on a requirement of share at one stage of a
// run, where the requirement's label marks k of the n valid cases. It
// panics when n is not positive or k is outside [0, n].
//
// Before the run is exhausted, the requirement is met when the lower bound
// reaches Tolerance times the share, and refuted when the upper bound
// falls below the share. An [Interim] check leaves any other requirement
// [Undecided]. A [Final] check of such a requirement, and every check of
// an [Exhausted] run, compare the observed share k/n with Tolerance times
// the share instead.
func Decide(k, n int, share float64, stage Stage) Verdict {
	checkShare(k, n)
	if stage != Exhausted {
		if Bound(k, n, -Z) >= Tolerance*share {
			return Met
		}
		if Bound(k, n, Z) < share {
			return Refuted
		}
		if stage == Interim {
			return Undecided
		}
	}
	if float64(k)/float64(n) >= Tolerance*share {
		return Met
	}
	return Unmet
}

// checkShare panics when k of n is no share: n is not positive, or k is
// outside [0, n].
func checkShare(k, n int) {
	if n < 1 || k < 0 || k > n {
		panic(fmt.Sprintf("coverage: %d of %d is not a share", k, n))
	}
}
