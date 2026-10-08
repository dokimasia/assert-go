// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import (
	"errors"
	"math"

	"go.dokimi.dev/assert/internal/fault"
)

// ErrRange reports integer bounds inside neither the signed nor the
// unsigned 64-bit range.
var ErrRange = errors.New("choice: integer bounds lie inside neither the signed nor the unsigned 64-bit range")

// maxInt is the largest value of the signed 64-bit range.
var maxInt = UintOf(math.MaxInt64)

// IntegerBounds are the inclusive bounds of an integer choice. Both lie
// inside the signed or both inside the unsigned 64-bit range, so the
// distance between any two values they admit fits in a uint64.
//
// IntegerBounds are comparable, and == reports whether two bounds are
// equal. The zero value is the bounds [0, 0].
type IntegerBounds struct {
	// lo and hi are the inclusive bounds.
	lo, hi Int
}

// NewIntegerBounds returns the bounds [lo, hi]. It returns [ErrEmpty] when
// lo exceeds hi, and [ErrRange] when lo is negative and hi is above
// math.MaxInt64, because no 64-bit range contains both.
func NewIntegerBounds(lo, hi Int) (IntegerBounds, error) {
	if lo.Compare(hi) > 0 {
		return IntegerBounds{}, fault.Of(ErrEmpty, "the integer bounds [%s, %s] admit no value", lo, hi)
	}
	if lo.Negative() && hi.Compare(maxInt) > 0 {
		return IntegerBounds{}, fault.Of(ErrRange, "the bounds [%s, %s] lie inside neither 64-bit range", lo, hi)
	}
	return IntegerBounds{lo: lo, hi: hi}, nil
}

// MustIntegerBounds returns the bounds [lo, hi] as [NewIntegerBounds]
// does, and panics with the error it returns. It is for bounds that are
// valid by construction, as [regexp.MustCompile] is for a literal pattern.
func MustIntegerBounds(lo, hi Int) IntegerBounds {
	b, err := NewIntegerBounds(lo, hi)
	if err != nil {
		panic(err)
	}
	return b
}

// Lo returns the inclusive lower bound. Every value the bounds admit is at
// least Lo.
func (b IntegerBounds) Lo() Int {
	return b.lo
}

// Hi returns the inclusive upper bound. Every value the bounds admit is at
// most Hi.
func (b IntegerBounds) Hi() Int {
	return b.hi
}

// Target returns the value closest to zero: zero when the bounds admit
// it, and otherwise the bound nearer zero.
func (b IntegerBounds) Target() Int {
	if b.hi.Negative() {
		return b.hi
	}
	if b.lo.Negative() {
		return Int{}
	}
	return b.lo
}

// Above returns the number of values the bounds admit above the target.
func (b IntegerBounds) Above() uint64 {
	return b.hi.Distance(b.Target())
}

// Below returns the number of values the bounds admit below the target.
func (b IntegerBounds) Below() uint64 {
	return b.Target().Distance(b.lo)
}

// Admits reports whether v lies inside the bounds.
func (b IntegerBounds) Admits(v Int) bool {
	return b.lo.Compare(v) <= 0 && v.Compare(b.hi) <= 0
}

// Coerce returns the recorded value when c is an integer the bounds
// admit, and the target otherwise.
func (b IntegerBounds) Coerce(c Choice) Int {
	if c.Kind == Integer && b.Admits(c.Integer) {
		return c.Integer
	}
	return b.Target()
}

// Key returns the sort key of v: its distance to the target, then 1 when v
// is below the target and 0 otherwise. Of two values at one distance, the
// one above the target is simpler. The key of a value the bounds do not
// admit is unspecified.
func (b IntegerBounds) Key(v Int) Key {
	target := b.Target()
	below := uint64(0)
	if v.Compare(target) < 0 {
		below = 1
	}
	return Key{kind: Integer, parts: [keyParts]uint64{v.Distance(target), below}}
}

// Rank returns the position of v in the key order of the bounds. The rank
// of a value the bounds do not admit is unspecified.
//
// The target is at 0. The order runs one above the target, one below, two
// above, two below, and so on. Past the bound of the shorter side, it
// continues one value at a time on the longer side, so every value of the
// bounds has a rank below their number of values.
func (b IntegerBounds) Rank(v Int) uint64 {
	target := b.Target()
	shorter := min(b.Above(), b.Below())
	distance := v.Distance(target)
	if distance > shorter {
		return shorter + distance
	}
	if v.Compare(target) > 0 {
		return 2*distance - 1
	}
	return 2 * distance
}

// AtRank returns the value at rank in the key order of the bounds. It is
// the inverse of [IntegerBounds.Rank] for every rank below the number of
// values the bounds admit. A larger rank returns a value outside the
// bounds.
func (b IntegerBounds) AtRank(rank uint64) Int {
	target := b.Target()
	below := b.Below()
	shorter := min(b.Above(), below)
	if rank > 2*shorter {
		distance := rank - shorter
		if below == shorter {
			return target.Add(distance)
		}
		return target.Sub(distance)
	}
	distance := rank/2 + rank%2
	if rank%2 == 1 {
		return target.Add(distance)
	}
	return target.Sub(distance)
}
