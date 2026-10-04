// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestWidth checks which values are widths, the fields of each format, and
// the assembly of a value from its fields.
func TestWidth(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Width
			want bool
		}{
			{name: "reports true for 32", give: choice.Width32, want: true},
			{name: "reports true for 64", give: choice.Width64, want: true},
			{name: "reports false for 0", give: 0, want: false},
			{name: "reports false for 16", give: 16, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a width")
			})
		}
	})

	t.Run("ExponentBits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Width
			want uint
		}{
			{name: "returns 8 for Width32", give: choice.Width32, want: 8},
			{name: "returns 11 for Width64", give: choice.Width64, want: 11},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.ExponentBits(), tt.want, "the bits of the biased exponent")
			})
		}
	})

	t.Run("MantissaBits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Width
			want uint
		}{
			{name: "returns 23 for Width32", give: choice.Width32, want: 23},
			{name: "returns 52 for Width64", give: choice.Width64, want: 52},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.MantissaBits(), tt.want, "the stored bits of the mantissa")
			})
		}
	})

	t.Run("FromParts", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name                     string
			w                        choice.Width
			sign, exponent, mantissa uint64
			want                     float64
		}{
			{name: "returns 1 at Width64", w: choice.Width64, sign: 0, exponent: 1023, mantissa: 0, want: 1},
			{
				name:     "returns -1.5 at Width64",
				w:        choice.Width64,
				sign:     1,
				exponent: 1023,
				mantissa: 1 << 51,
				want:     -1.5,
			},
			{
				name:     "returns -0 at Width64",
				w:        choice.Width64,
				sign:     1,
				exponent: 0,
				mantissa: 0,
				want:     negativeZero,
			},
			{name: "returns 1 at Width32", w: choice.Width32, sign: 0, exponent: 127, mantissa: 0, want: 1},
			{
				name:     "returns -0.75 at Width32",
				w:        choice.Width32,
				sign:     1,
				exponent: 126,
				mantissa: 1 << 22,
				want:     -0.75,
			},
			{
				name:     "returns the smallest subnormal at Width32",
				w:        choice.Width32,
				sign:     0,
				exponent: 0,
				mantissa: 1,
				want:     math.SmallestNonzeroFloat32,
			},
			{
				name:     "returns positive infinity at Width32",
				w:        choice.Width32,
				sign:     0,
				exponent: 255,
				mantissa: 0,
				want:     math.Inf(1),
			},
			{
				name:     "returns NaN at Width64",
				w:        choice.Width64,
				sign:     0,
				exponent: 2047,
				mantissa: 1,
				want:     math.NaN(),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.w.FromParts(tt.sign, tt.exponent, tt.mantissa)
				assert.True(t, choice.SameFloat(got, tt.want), "the assembled value")
			})
		}
	})
}

// TestWidthAllocs checks that no method of Width allocates.
func TestWidthAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.Width32.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.Width32.ExponentBits() }, 0, "ExponentBits allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.Width32.MantissaBits() }, 0, "MantissaBits allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.Width32.FromParts(1, 126, 1<<22) }, 0, "FromParts allocates nothing")
}

// BenchmarkWidth measures each method of Width under a ceiling of no
// allocation.
func BenchmarkWidth(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.Width64.Valid()
		}
		assert.True(b, got, "64 is a width")
	})

	b.Run("ExponentBits", func(b *testing.B) {
		var got uint
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.Width32.ExponentBits()
		}
		assert.Equal(b, got, uint(8), "the bits of a binary32 exponent")
	})

	b.Run("MantissaBits", func(b *testing.B) {
		var got uint
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.Width32.MantissaBits()
		}
		assert.Equal(b, got, uint(23), "the stored bits of a binary32 mantissa")
	})

	b.Run("FromParts", func(b *testing.B) {
		var got float64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.Width32.FromParts(1, 126, 1<<22)
		}
		assert.Equal(b, got, -0.75, "the assembled value")
	})
}
