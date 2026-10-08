// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import (
	"math"
	"math/bits"
)

// NaNBits are the bits of the canonical quiet NaN. Every NaN in a choice
// has them.
const NaNBits = 0x7FF8000000000000

// The fields of a binary64 float.
const (
	// exponentMask selects the biased exponent after a shift by
	// mantissaBits64: eleven bits.
	exponentMask = 0x7FF
	// mantissaMask selects the stored mantissa: the low 52 bits.
	mantissaMask = 0xFFFFFFFFFFFFF
	// subnormalScale is 1074, the number of fractional bits of the
	// smallest subnormal. A mantissa at the smallest biased exponent
	// counts units of 2^-1074.
	subnormalScale = 1074
)

// NaN returns the canonical quiet NaN, the float with the bits [NaNBits].
// It differs from math.NaN, whose payload is 1.
func NaN() float64 {
	return math.Float64frombits(NaNBits)
}

// Representable reports whether x is a value of a float of width w. Every
// float64 is a value of width 64, and NaN and the infinities are values of
// both widths.
func Representable(x float64, w Width) bool {
	if w == Width64 || math.IsNaN(x) || math.IsInf(x, 0) {
		return true
	}
	return float64(float32(x)) == x
}

// NextUp returns the smallest value of width w above x, for x a value of
// the width. Both zeros step to the smallest positive subnormal, and
// positive infinity and NaN return themselves.
func NextUp(x float64, w Width) float64 {
	if w == Width64 {
		return math.Nextafter(x, math.Inf(1))
	}
	if math.IsNaN(x) || math.IsInf(x, 1) {
		return x
	}
	if x == 0 {
		return float64(math.Float32frombits(1))
	}
	raw := math.Float32bits(float32(x))
	if math.Signbit(x) {
		return float64(math.Float32frombits(raw - 1))
	}
	return float64(math.Float32frombits(raw + 1))
}

// NextDown returns the largest value of width w below x, for x a value of
// the width. Both zeros step to the negative subnormal nearest zero, and
// negative infinity and NaN return themselves.
func NextDown(x float64, w Width) float64 {
	return -NextUp(-x, w)
}

// SameFloat reports whether a and b are one value. Every NaN is one value,
// whatever its payload, and -0 differs from +0.
func SameFloat(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return math.Float64bits(a) == math.Float64bits(b)
}

// FractionBits returns the number of bits after the binary point of a
// finite x: the f of |x| in lowest terms n / 2^f. An integer has none.
func FractionBits(x float64) uint64 {
	if x == 0 {
		return 0
	}
	raw := math.Float64bits(math.Abs(x))
	exponent := int(raw >> mantissaBits64 & exponentMask)
	mantissa := raw & mantissaMask
	if exponent == 0 {
		exponent = 1
	} else {
		mantissa |= 1 << mantissaBits64
	}
	lowest := exponent - 1 - subnormalScale + bits.TrailingZeros64(mantissa)
	return uint64(max(-lowest, 0))
}
