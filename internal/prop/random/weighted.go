// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random

// Weighted draws an index into weights with probability proportional to its
// weight. One weight returns 0 and consumes nothing. Otherwise the draw is
// one Below(the sum of the weights), and the index is the first whose
// running sum of weights exceeds it. Every weight is 1 or more, and their
// sum is below 2^64.
func Weighted(s *Source, weights []uint64) int {
	if len(weights) == 1 {
		return 0
	}
	var sum uint64
	for _, w := range weights {
		sum += w
	}
	point, index := s.Below(sum), 0
	for point >= weights[index] {
		point -= weights[index]
		index++
	}
	return index
}
