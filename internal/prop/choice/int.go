// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import (
	"cmp"
	"math"
	"strconv"
)

// Int is the value of an integer choice: any integer from -2^63 to
// 2^64 - 1, the union of the signed and the unsigned 64-bit ranges.
//
// An Int keeps its sign whatever bounds requested it, so -5 and 2^64 - 5
// are different values. The zero value is 0. Int is comparable, and ==
// reports whether two values are equal.
type Int struct {
	// magnitude is the absolute value.
	magnitude uint64
	// negative is set only with a magnitude from 1 to 2^63, so that every
	// value has one representation.
	negative bool
}

// IntOf returns v as an [Int].
func IntOf(v int64) Int {
	if v < 0 {
		return Int{magnitude: -uint64(v), negative: true}
	}
	return Int{magnitude: uint64(v)}
}

// UintOf returns v as an [Int].
func UintOf(v uint64) Int {
	return Int{magnitude: v}
}

// Int64 returns i as an int64. It reports false, with 0, when i is above
// math.MaxInt64.
func (i Int) Int64() (int64, bool) {
	if i.negative {
		return int64(-i.magnitude), true
	}
	if i.magnitude > math.MaxInt64 {
		return 0, false
	}
	return int64(i.magnitude), true
}

// Uint64 returns i as a uint64. It reports false, with 0, when i is
// negative.
func (i Int) Uint64() (uint64, bool) {
	if i.negative {
		return 0, false
	}
	return i.magnitude, true
}

// Float64 returns the float64 nearest to i. Every integer of magnitude
// below 2^53 converts exactly.
func (i Int) Float64() float64 {
	if i.negative {
		return -float64(i.magnitude)
	}
	return float64(i.magnitude)
}

// Negative reports whether i is below zero.
func (i Int) Negative() bool {
	return i.negative
}

// Magnitude returns the absolute value of i, from 0 to 2^64 - 1.
func (i Int) Magnitude() uint64 {
	return i.magnitude
}

// Compare returns -1 when i is less than j, 0 when they are equal, and +1
// when i is greater.
func (i Int) Compare(j Int) int {
	if i.negative != j.negative {
		if i.negative {
			return -1
		}
		return 1
	}
	order := cmp.Compare(i.magnitude, j.magnitude)
	if i.negative {
		return -order
	}
	return order
}

// Add returns i + d. The sum must not exceed 2^64 - 1.
func (i Int) Add(d uint64) Int {
	if !i.negative {
		return Int{magnitude: i.magnitude + d}
	}
	if d < i.magnitude {
		return Int{magnitude: i.magnitude - d, negative: true}
	}
	return Int{magnitude: d - i.magnitude}
}

// Sub returns i - d. The difference must not be below -2^63.
func (i Int) Sub(d uint64) Int {
	if i.negative {
		return Int{magnitude: i.magnitude + d, negative: true}
	}
	if d <= i.magnitude {
		return Int{magnitude: i.magnitude - d}
	}
	return Int{magnitude: d - i.magnitude, negative: true}
}

// Distance returns |i - j|, for two values inside one 64-bit range, signed
// or unsigned, so that the distance fits in a uint64.
func (i Int) Distance(j Int) uint64 {
	if i.negative != j.negative {
		return i.magnitude + j.magnitude
	}
	return max(i.magnitude, j.magnitude) - min(i.magnitude, j.magnitude)
}

// String returns i in decimal, with a leading minus sign when i is
// negative.
func (i Int) String() string {
	if i.negative {
		return "-" + strconv.FormatUint(i.magnitude, 10)
	}
	return strconv.FormatUint(i.magnitude, 10)
}
