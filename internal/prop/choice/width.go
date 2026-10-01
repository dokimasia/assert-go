// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import "math"

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Width -linecomment -output=width.string_gen.go

// The fields and the exact integers of the two formats.
const (
	// exponentBits32 is the number of bits of a binary32 biased exponent.
	exponentBits32 = 8
	// mantissaBits32 is the number of stored bits of a binary32 mantissa.
	mantissaBits32 = 23
	// exponentBits64 is the number of bits of a binary64 biased exponent.
	exponentBits64 = 11
	// mantissaBits64 is the number of stored bits of a binary64 mantissa.
	mantissaBits64 = 52
	// integerLimit32 is 2^24, the magnitude below which every integer is a
	// binary32 value.
	integerLimit32 = 16777216
	// integerLimit64 is 2^53, the magnitude below which every integer is a
	// binary64 value.
	integerLimit64 = 9007199254740992
)

// Width is the width of a float choice in bits: the IEEE 754 format that
// every value of the choice has.
type Width uint8

const (
	// Width32 is the binary32 format.
	Width32 Width = 32 // 32
	// Width64 is the binary64 format.
	Width64 Width = 64 // 64
)

// Valid reports whether w is one of the two widths.
func (w Width) Valid() bool {
	return w == Width32 || w == Width64
}

// ExponentBits returns the number of bits of the biased exponent of a
// value of width w: 8 at Width32 and 11 at Width64.
func (w Width) ExponentBits() uint {
	if w == Width64 {
		return exponentBits64
	}
	return exponentBits32
}

// MantissaBits returns the number of stored bits of the mantissa of a
// value of width w: 23 at Width32 and 52 at Width64.
func (w Width) MantissaBits() uint {
	if w == Width64 {
		return mantissaBits64
	}
	return mantissaBits32
}

// FromParts returns the value of width w with the given sign bit, biased
// exponent and stored mantissa. Each part must fit its field: the sign
// below 2, the exponent below 2^ExponentBits and the mantissa below
// 2^MantissaBits.
func (w Width) FromParts(sign, exponent, mantissa uint64) float64 {
	bits := (sign<<w.ExponentBits()|exponent)<<w.MantissaBits() | mantissa
	if w == Width64 {
		return math.Float64frombits(bits)
	}
	return float64(math.Float32frombits(uint32(bits)))
}

// integerLimit returns the magnitude below which every integer is a value
// of width w: 2^24 at Width32 and 2^53 at Width64.
func (w Width) integerLimit() uint64 {
	if w == Width64 {
		return integerLimit64
	}
	return integerLimit32
}
