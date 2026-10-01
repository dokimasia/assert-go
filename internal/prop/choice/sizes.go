// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import "fmt"

// Sizes are the bounds of a length: an inclusive minimum, and an inclusive
// maximum or none.
//
// Without a maximum the length is unbounded, and the cap on the choices of
// a case bounds it instead. The zero value admits every length. Sizes are
// comparable, and == reports whether two bounds are equal.
type Sizes struct {
	// minSize is the shortest length.
	minSize int
	// maxSize is the longest length when bounded is set, and 0 otherwise.
	maxSize int
	// bounded reports whether maxSize applies.
	bounded bool
}

// NewSizes returns the lengths from minSize to maxSize. It returns
// [ErrEmpty] when minSize is negative or maxSize is below it.
func NewSizes(minSize, maxSize int) (Sizes, error) {
	if maxSize < minSize {
		return Sizes{}, fmt.Errorf("%w: sizes [%d, %d]", ErrEmpty, minSize, maxSize)
	}
	s, err := NewUnboundedSizes(minSize)
	if err != nil {
		return Sizes{}, err
	}
	s.maxSize, s.bounded = maxSize, true
	return s, nil
}

// NewUnboundedSizes returns the lengths of minSize or more. It returns
// [ErrEmpty] when minSize is negative.
func NewUnboundedSizes(minSize int) (Sizes, error) {
	if minSize < 0 {
		return Sizes{}, fmt.Errorf("%w: minimum size %d", ErrEmpty, minSize)
	}
	return Sizes{minSize: minSize}, nil
}

// Min returns the shortest length.
func (s Sizes) Min() int {
	return s.minSize
}

// Max returns the longest length. It reports false when the length is
// unbounded.
func (s Sizes) Max() (int, bool) {
	return s.maxSize, s.bounded
}

// Admits reports whether n is a length of the sizes.
func (s Sizes) Admits(n int) bool {
	return n >= s.minSize && (!s.bounded || n <= s.maxSize)
}

// Clamp returns n when the sizes admit it, and otherwise the bound nearer
// to n.
func (s Sizes) Clamp(n int) int {
	if s.bounded {
		n = min(n, s.maxSize)
	}
	return max(n, s.minSize)
}

// FlagBounds returns the bounds of the decision whether a collection of
// count elements gets another, where 1 continues it and 0 stops it: [1, 1]
// below the minimum, [0, 0] at the maximum, and [0, 1] otherwise.
func (s Sizes) FlagBounds(count int) IntegerBounds {
	if count < s.minSize {
		return IntegerBounds{lo: UintOf(1), hi: UintOf(1)}
	}
	if s.bounded && count >= s.maxSize {
		return IntegerBounds{}
	}
	return IntegerBounds{hi: UintOf(1)}
}
