// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import "slices"

// Choice is one decision of a case: its kind and the value the case used.
//
// The field the kind names contains the value, and the others are zero.
type Choice struct {
	// Kind is the kind of the choice.
	Kind Kind
	// Integer is the value of an integer choice.
	Integer Int
	// Float is the value of a float choice. A NaN has the bits [NaNBits].
	Float float64
	// Sequence is the value of a sequence choice. A choice shares the
	// slice it was given, and nothing modifies the slice after.
	Sequence []uint32
}

// Equal reports whether c and d are one choice: the same kind and the
// same value. Floats compare by their bits, so -0 differs from +0, and
// every NaN equals every other.
func (c Choice) Equal(d Choice) bool {
	if c.Kind != d.Kind {
		return false
	}
	if c.Kind == Integer {
		return c.Integer == d.Integer
	}
	if c.Kind == Float {
		return SameFloat(c.Float, d.Float)
	}
	return slices.Equal(c.Sequence, d.Sequence)
}
