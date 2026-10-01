// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import (
	"cmp"
	"math"
	"slices"
)

// keyParts is the number of fixed numbers in a [Key].
const keyParts = 4

// The groups of a float's sort key, from the simplest.
const (
	// groupFinite contains every finite value.
	groupFinite uint64 = 0
	// groupInfinite contains the two infinities.
	groupInfinite uint64 = 1
	// groupNaN contains NaN.
	groupNaN uint64 = 2
)

// Key is the sort key of a choice under its bounds. Of two keys, the
// smaller is the simpler value, and the shrinker keeps a candidate only
// when its keys are shortlex-smaller than the best case's.
//
// Keys of different kinds order by [Kind]. An integer's key is its
// distance to the target, then whether it lies below the target, as
// [IntegerBounds.Key] states. A float's key is [FloatKey], and a
// sequence's key is [SequenceKey]. The zero value is the key of an
// integer at its target.
type Key struct {
	// kind is the kind of the choice, compared first.
	kind Kind
	// parts are the key's fixed numbers, compared in order, with unused
	// places zero.
	parts [keyParts]uint64
	// sequence is the elements of a sequence choice, compared after parts,
	// whose first number is the sequence's length. The key shares it.
	sequence []uint32
}

// FloatKey returns the sort key of x. Finite values come first, ordered by
// their number of fractional bits and then by magnitude, so every integer
// precedes every fraction. The infinities follow, and NaN is last. At
// equal magnitude, a positive value precedes the negative one, and +0
// precedes -0.
func FloatKey(x float64) Key {
	if math.IsNaN(x) {
		return Key{kind: Float, parts: [keyParts]uint64{groupNaN}}
	}
	negative := uint64(0)
	if math.Signbit(x) {
		negative = 1
	}
	if math.IsInf(x, 0) {
		return Key{kind: Float, parts: [keyParts]uint64{groupInfinite, negative}}
	}
	magnitude := math.Abs(x)
	return Key{
		kind:  Float,
		parts: [keyParts]uint64{groupFinite, FractionBits(magnitude), math.Float64bits(magnitude), negative},
	}
}

// SequenceKey returns the sort key of s: its length, then its elements in
// order. A shorter sequence is simpler, and of two equally long ones, the
// one smaller at the first element where they differ. The key shares s.
func SequenceKey(s []uint32) Key {
	return Key{kind: Sequence, parts: [keyParts]uint64{uint64(len(s))}, sequence: s}
}

// Compare returns -1 when k is simpler than o, 0 when they are equally
// simple, and +1 when o is simpler.
func (k Key) Compare(o Key) int {
	if c := cmp.Compare(k.kind, o.kind); c != 0 {
		return c
	}
	for i := range k.parts {
		if c := cmp.Compare(k.parts[i], o.parts[i]); c != 0 {
			return c
		}
	}
	return slices.Compare(k.sequence, o.sequence)
}
