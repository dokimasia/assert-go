// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package alphabet

import (
	"cmp"
	"slices"
)

// Interval is an inclusive range of indices of the default alphabet. An
// interval whose First exceeds its Last contains no index.
type Interval struct {
	// First is the smallest index of the interval.
	First uint32
	// Last is the largest index of the interval.
	Last uint32
}

// Merge returns the union of intervals as the fewest intervals, sorted by
// First, that neither overlap nor touch. It sorts intervals in place and
// returns a prefix of it, so the caller's slice changes.
func Merge(intervals []Interval) []Interval {
	slices.SortFunc(intervals, func(a, b Interval) int { return cmp.Compare(a.First, b.First) })
	merged := intervals[:0]
	for _, v := range intervals {
		if n := len(merged); n > 0 && v.First <= merged[n-1].Last+1 {
			merged[n-1].Last = max(merged[n-1].Last, v.Last)
			continue
		}
		merged = append(merged, v)
	}
	return merged
}
