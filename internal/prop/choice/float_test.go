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

// The extreme binary32 values, as float64.
var (
	// maxFloat32 is the largest finite binary32 value.
	maxFloat32 = float64(math.MaxFloat32)
	// minFloat32 is the smallest positive binary32 subnormal.
	minFloat32 = float64(math.Float32frombits(1))
)

// negativeZero is -0.
var negativeZero = math.Copysign(0, -1)

// TestFloat checks the functions over float values: the canonical NaN,
// which values a width has, the steps between them, the fractional bits,
// and value identity.
func TestFloat(t *testing.T) {
	t.Parallel()

	t.Run("NaN", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the float with NaNBits", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, math.Float64bits(choice.NaN()), uint64(choice.NaNBits), "the canonical bits")
		})
	})

	t.Run("Representable", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			x    float64
			w    choice.Width
			want bool
		}{
			{name: "reports true for 0.1 at width 64", x: 0.1, w: choice.Width64, want: true},
			{name: "reports false for 0.1 at width 32", x: 0.1, w: choice.Width32, want: false},
			{name: "reports true for 0.5 at width 32", x: 0.5, w: choice.Width32, want: true},
			{name: "reports false for 1e300 at width 32", x: 1e300, w: choice.Width32, want: false},
			{name: "reports true for infinity at width 32", x: math.Inf(-1), w: choice.Width32, want: true},
			{name: "reports true for NaN at width 32", x: math.NaN(), w: choice.Width32, want: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, choice.Representable(tt.x, tt.w), tt.want, "whether x is a value of the width")
			})
		}
	})

	t.Run("NextUp", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			x    float64
			w    choice.Width
			want float64
		}{
			{name: "steps 1 by one unit at width 32", x: 1, w: choice.Width32, want: 1 + 0x1p-23},
			{name: "steps +0 to the smallest subnormal at width 32", x: 0, w: choice.Width32, want: minFloat32},
			{
				name: "steps -0 to the smallest subnormal at width 32",
				x:    negativeZero,
				w:    choice.Width32,
				want: minFloat32,
			},
			{
				name: "steps the negative subnormal nearest zero to -0 at width 32",
				x:    -minFloat32,
				w:    choice.Width32,
				want: negativeZero,
			},
			{name: "steps the largest binary32 to infinity", x: maxFloat32, w: choice.Width32, want: math.Inf(1)},
			{
				name: "steps negative infinity to the most negative binary32",
				x:    math.Inf(-1),
				w:    choice.Width32,
				want: -maxFloat32,
			},
			{
				name: "returns positive infinity unchanged at width 32",
				x:    math.Inf(1),
				w:    choice.Width32,
				want: math.Inf(1),
			},
			{
				name: "steps +0 to the smallest subnormal at width 64",
				x:    0,
				w:    choice.Width64,
				want: math.SmallestNonzeroFloat64,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := choice.NextUp(tt.x, tt.w)
				assert.True(t, choice.SameFloat(got, tt.want), "the next value of the width")
			})
		}

		t.Run("returns NaN for NaN at width 32", func(t *testing.T) {
			t.Parallel()
			assert.True(t, math.IsNaN(choice.NextUp(math.NaN(), choice.Width32)), "NaN has no next value")
		})
	})

	t.Run("NextDown", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			x    float64
			w    choice.Width
			want float64
		}{
			{
				name: "steps +0 to the negative subnormal nearest zero at width 32",
				x:    0,
				w:    choice.Width32,
				want: -minFloat32,
			},
			{name: "steps 1 by half a unit at width 64", x: 1, w: choice.Width64, want: 1 - 0x1p-53},
			{
				name: "returns negative infinity unchanged at width 32",
				x:    math.Inf(-1),
				w:    choice.Width32,
				want: math.Inf(-1),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := choice.NextDown(tt.x, tt.w)
				assert.True(t, choice.SameFloat(got, tt.want), "the previous value of the width")
			})
		}
	})

	t.Run("FractionBits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give float64
			want uint64
		}{
			{name: "returns 0 for zero", give: 0, want: 0},
			{name: "returns 0 for an integer", give: -12, want: 0},
			{name: "returns 0 for the largest binary64", give: math.MaxFloat64, want: 0},
			{name: "returns 1 for a half", give: -2.5, want: 1},
			{name: "returns 3 for 0.625", give: 0.625, want: 3},
			{name: "returns 1074 for the smallest subnormal", give: math.SmallestNonzeroFloat64, want: 1074},
			{name: "returns 1022 for the smallest normal", give: 0x1p-1022, want: 1022},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, choice.FractionBits(tt.give), tt.want, "the bits after the binary point")
			})
		}
	})

	t.Run("SameFloat", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			a, b float64
			want bool
		}{
			{name: "reports false for +0 against -0", a: 0, b: negativeZero, want: false},
			{name: "reports true for two NaN payloads", a: math.NaN(), b: choice.NaN(), want: true},
			{name: "reports false for NaN against 0", a: choice.NaN(), b: 0, want: false},
			{name: "reports false for 0 against NaN", a: 0, b: choice.NaN(), want: false},
			{name: "reports true for one value", a: 0.25, b: 0.25, want: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, choice.SameFloat(tt.a, tt.b), tt.want, "whether the two floats are one value")
			})
		}
	})
}

// TestFloatZeroAlloc checks that no function over float values allocates.
func TestFloatZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.NaN() }, 0, "NaN allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.Representable(0.5, choice.Width32) }, 0, "Representable allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.NextUp(1, choice.Width32) }, 0, "NextUp allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.NextDown(1, choice.Width32) }, 0, "NextDown allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.FractionBits(0.625) }, 0, "FractionBits allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.SameFloat(0, negativeZero) }, 0, "SameFloat allocates nothing")
}

// BenchmarkFloat measures each function over float values under a ceiling
// of no allocation.
func BenchmarkFloat(b *testing.B) {
	b.Run("NaN", func(b *testing.B) {
		var got float64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.NaN()
		}
		assert.True(b, math.IsNaN(got), "the canonical NaN")
	})

	b.Run("Representable", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.Representable(0.1, choice.Width32)
		}
		assert.False(b, got, "0.1 is not a binary32 value")
	})

	b.Run("NextUp", func(b *testing.B) {
		var got float64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.NextUp(1, choice.Width32)
		}
		assert.Equal(b, got, 1+0x1p-23, "the next binary32 value above 1")
	})

	b.Run("NextDown", func(b *testing.B) {
		var got float64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.NextDown(1, choice.Width64)
		}
		assert.Equal(b, got, 1-0x1p-53, "the next binary64 value below 1")
	})

	b.Run("FractionBits", func(b *testing.B) {
		var got uint64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.FractionBits(0.625)
		}
		assert.Equal(b, got, uint64(3), "the bits of 5/8")
	})

	b.Run("SameFloat", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.SameFloat(0, negativeZero)
		}
		assert.False(b, got, "-0 is not +0")
	})
}
