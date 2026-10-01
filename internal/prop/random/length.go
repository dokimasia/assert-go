// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random

import "go.dokimi.dev/assert/internal/prop/choice"

// defaultExtra is the average number of elements beyond its minimum that a
// collection gets when its sizes allow that many.
const defaultExtra = 5

// Average returns the average length that the random phase aims for under
// sizes: min + min(max(min, 5), ceil((max - min) / 2)), and
// min + max(min, 5) without a maximum. The average is an integer, so the
// continue coin of [Flag] has integer odds.
func Average(sizes choice.Sizes) int {
	extra := max(sizes.Min(), defaultExtra)
	if maxSize, bounded := sizes.Max(); bounded {
		extra = min(extra, (maxSize-sizes.Min()+1)/2)
	}
	return sizes.Min() + extra
}

// Flag decides whether a collection of count elements gets another
// element. A decision that [choice.Sizes.FlagBounds] forces returns its one
// value, true for 1, and consumes nothing. A free decision continues by a
// Coin(extra, extra + 1), where extra is the [Average] length minus the
// minimum.
func Flag(s *Source, sizes choice.Sizes, count int) bool {
	bounds := sizes.FlagBounds(count)
	if lo := bounds.Lo(); lo == bounds.Hi() {
		return lo == choice.UintOf(1)
	}
	extra := uint64(Average(sizes) - sizes.Min())
	return s.Coin(extra, extra+1)
}
