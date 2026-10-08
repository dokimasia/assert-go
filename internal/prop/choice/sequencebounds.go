// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import "go.dokimi.dev/assert/internal/fault"

// SequenceBounds are the bounds of a sequence choice: the range [0, k) of
// every element, and the [Sizes] of the length.
//
// The constructor never returns the zero value, whose k of 0 admits only
// the empty sequence. SequenceBounds are comparable, and == reports
// whether two bounds are equal.
type SequenceBounds struct {
	// k is one more than the largest element.
	k uint32
	// sizes bounds the length.
	sizes Sizes
}

// NewSequenceBounds returns the bounds of sequences of integers in [0, k)
// whose lengths sizes admits. It returns [ErrEmpty] when k is 0.
func NewSequenceBounds(k uint32, sizes Sizes) (SequenceBounds, error) {
	if k == 0 {
		return SequenceBounds{}, fault.Of(ErrEmpty, "the elements [0, 0) admit no value")
	}
	return SequenceBounds{k: k, sizes: sizes}, nil
}

// MustSequenceBounds returns the bounds as [NewSequenceBounds] does, and
// panics with the error it returns. It is for bounds that are valid by
// construction.
func MustSequenceBounds(k uint32, sizes Sizes) SequenceBounds {
	b, err := NewSequenceBounds(k, sizes)
	if err != nil {
		panic(err)
	}
	return b
}

// K returns one more than the largest element the bounds admit.
func (b SequenceBounds) K() uint32 {
	return b.k
}

// Sizes returns the bounds of the length.
func (b SequenceBounds) Sizes() Sizes {
	return b.sizes
}

// Element returns the bounds of one element, [0, K - 1].
func (b SequenceBounds) Element() IntegerBounds {
	return IntegerBounds{hi: UintOf(uint64(b.k) - 1)}
}

// Target returns the simplest sequence: as many zeros as the minimum
// length. It allocates them, and returns nil for a minimum of zero.
func (b SequenceBounds) Target() []uint32 {
	if b.sizes.Min() == 0 {
		return nil
	}
	return make([]uint32, b.sizes.Min())
}

// Admits reports whether s is a sequence of the bounds: a length that the
// sizes admit, and every element below K.
func (b SequenceBounds) Admits(s []uint32) bool {
	if !b.sizes.Admits(len(s)) {
		return false
	}
	for _, element := range s {
		if element >= b.k {
			return false
		}
	}
	return true
}

// Coerce returns the recorded sequence fitted to the bounds, and the
// target when c is of another kind. Fitting makes three changes:
//
//   - A sequence longer than the maximum is cut to the maximum.
//   - A sequence shorter than the minimum is extended with zeros.
//   - An element of K or more becomes 0.
//
// Coerce returns a sequence the bounds admit as the slice c shares,
// without an allocation. Any other result is a new slice.
func (b SequenceBounds) Coerce(c Choice) []uint32 {
	if c.Kind != Sequence {
		return b.Target()
	}
	if b.Admits(c.Sequence) {
		return c.Sequence
	}
	fitted := make([]uint32, b.sizes.Clamp(len(c.Sequence)))
	for i, element := range c.Sequence[:min(len(c.Sequence), len(fitted))] {
		if element < b.k {
			fitted[i] = element
		}
	}
	return fitted
}
