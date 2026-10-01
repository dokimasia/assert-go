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

// TestFloatBounds checks the construction, the target, the admission, the
// replay and the exact integers of float bounds, for both widths and both
// NaN policies.
func TestFloatBounds(t *testing.T) {
	t.Parallel()

	t.Run("NewFloatBounds", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			lo, hi float64
			nan    choice.NaNPolicy
			w      choice.Width
			want   error
		}{
			{
				name: "returns ErrWidth for a width of 16",
				lo:   0,
				hi:   1,
				nan:  choice.ExcludeNaN,
				w:    16,
				want: choice.ErrWidth,
			},
			{
				name: "returns ErrNaNPolicy for a policy past the two",
				lo:   0,
				hi:   1,
				nan:  invalidNaNPolicy,
				w:    choice.Width64,
				want: choice.ErrNaNPolicy,
			},
			{
				name: "returns ErrEmpty for a NaN lower bound",
				lo:   math.NaN(),
				hi:   1,
				nan:  choice.ExcludeNaN,
				w:    choice.Width64,
				want: choice.ErrEmpty,
			},
			{
				name: "returns ErrEmpty for a NaN upper bound",
				lo:   0,
				hi:   math.NaN(),
				nan:  choice.ExcludeNaN,
				w:    choice.Width64,
				want: choice.ErrEmpty,
			},
			{
				name: "returns ErrEmpty when lo exceeds hi",
				lo:   1,
				hi:   0,
				nan:  choice.ExcludeNaN,
				w:    choice.Width64,
				want: choice.ErrEmpty,
			},
			{
				name: "returns ErrNotOfWidth for a lower bound of 0.1 at width 32",
				lo:   0.1,
				hi:   1,
				nan:  choice.ExcludeNaN,
				w:    choice.Width32,
				want: choice.ErrNotOfWidth,
			},
			{
				name: "returns ErrNotOfWidth for an upper bound of 0.1 at width 32",
				lo:   0,
				hi:   0.1,
				nan:  choice.ExcludeNaN,
				w:    choice.Width32,
				want: choice.ErrNotOfWidth,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := choice.NewFloatBounds(tt.lo, tt.hi, tt.nan, tt.w)
				assert.ErrorIs(t, err, tt.want, "the refusal")
			})
		}

		t.Run("returns bounds that report what they were given", func(t *testing.T) {
			t.Parallel()
			b := floatBounds(t, -1, 2, choice.AdmitNaN, choice.Width32)
			assert.Equal(t, [2]float64{b.Lo(), b.Hi()}, [2]float64{-1, 2}, "the range")
			assert.Equal(t, b.NaNPolicy(), choice.AdmitNaN, "the NaN policy")
			assert.Equal(t, b.Width(), choice.Width32, "the width")
		})
	})

	t.Run("Target", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			lo, hi float64
			w      choice.Width
			want   float64
		}{
			{name: "returns 0 when the range straddles it", lo: -1, hi: 1, w: choice.Width64, want: 0},
			{name: "returns 0 when it is the lower bound", lo: 0, hi: 2.5, w: choice.Width64, want: 0},
			{name: "returns 0 when it is the upper bound", lo: -2.5, hi: 0, w: choice.Width64, want: 0},
			{name: "returns +0 for a lower bound of -0", lo: negativeZero, hi: 1, w: choice.Width64, want: 0},
			{name: "returns +0 for an upper bound of -0", lo: -1, hi: negativeZero, w: choice.Width64, want: 0},
			{name: "returns the smallest integer in a positive range", lo: 2.5, hi: 7, w: choice.Width64, want: 3},
			{name: "returns an odd integer at the upper bound", lo: 0.5, hi: 1, w: choice.Width64, want: 1},
			{name: "returns an even integer at the upper bound", lo: 1.5, hi: 2, w: choice.Width64, want: 2},
			{name: "returns a fraction at the upper bound", lo: 0.6, hi: 0.75, w: choice.Width64, want: 0.75},
			{
				name: "returns the integer nearest zero in a negative range",
				lo:   -7,
				hi:   -2.5,
				w:    choice.Width64,
				want: -3,
			},
			{name: "returns the fraction of fewest bits", lo: 0.1, hi: 0.9, w: choice.Width64, want: 0.5},
			{
				name: "returns the fraction of fewest bits when one bit is not enough",
				lo:   0.6,
				hi:   0.7,
				w:    choice.Width64,
				want: 0.625,
			},
			{
				name: "returns the fraction of fewest bits at width 32",
				lo:   0.5625,
				hi:   0.6875,
				w:    choice.Width32,
				want: 0.625,
			},
			{
				name: "returns the lower bound when no simpler value is inside",
				lo:   0.6875,
				hi:   0.6875,
				w:    choice.Width32,
				want: 0.6875,
			},
			{
				name: "returns the smallest integer at or above 2^53",
				lo:   1<<53 + 2,
				hi:   1 << 60,
				w:    choice.Width64,
				want: 1<<53 + 2,
			},
			{
				name: "returns the smallest integer of a range that crosses 2^53",
				lo:   1<<52 - 0.5,
				hi:   1 << 60,
				w:    choice.Width64,
				want: 1 << 52,
			},
			{
				name: "returns the lower bound for a range up to infinity",
				lo:   1e300,
				hi:   math.Inf(1),
				w:    choice.Width64,
				want: 1e300,
			},
			{
				name: "returns positive infinity for bounds of it alone",
				lo:   math.Inf(1),
				hi:   math.Inf(1),
				w:    choice.Width64,
				want: math.Inf(1),
			},
			{
				name: "returns negative infinity for bounds of it alone",
				lo:   math.Inf(-1),
				hi:   math.Inf(-1),
				w:    choice.Width64,
				want: math.Inf(-1),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := floatBounds(t, tt.lo, tt.hi, choice.ExcludeNaN, tt.w).Target()
				assert.True(t, choice.SameFloat(got, tt.want), "the simplest value of the bounds")
			})
		}
	})

	t.Run("Admits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			b    choice.FloatBounds
			give float64
			want bool
		}{
			{
				name: "reports true for NaN when the bounds admit it",
				b:    floatBounds(t, 0, 1, choice.AdmitNaN, choice.Width64),
				give: math.NaN(),
				want: true,
			},
			{
				name: "reports false for NaN when the bounds exclude it",
				b:    floatBounds(t, 0, 1, choice.ExcludeNaN, choice.Width64),
				give: math.NaN(),
				want: false,
			},
			{
				name: "reports true for a value of the width inside the range",
				b:    floatBounds(t, 1, 2, choice.ExcludeNaN, choice.Width32),
				give: 1.5,
				want: true,
			},
			{
				name: "reports false for a value of another width",
				b:    floatBounds(t, 1, 2, choice.ExcludeNaN, choice.Width32),
				give: 1.1,
				want: false,
			},
			{
				name: "reports false below the range",
				b:    floatBounds(t, 1, 2, choice.ExcludeNaN, choice.Width64),
				give: 0.5,
				want: false,
			},
			{
				name: "reports false above the range",
				b:    floatBounds(t, 1, 2, choice.ExcludeNaN, choice.Width64),
				give: 3,
				want: false,
			},
			{
				name: "reports true for the lower bound",
				b:    floatBounds(t, 1, 2, choice.ExcludeNaN, choice.Width64),
				give: 1,
				want: true,
			},
			{
				name: "reports true for the upper bound",
				b:    floatBounds(t, 1, 2, choice.ExcludeNaN, choice.Width64),
				give: 2,
				want: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.b.Admits(tt.give), tt.want, "whether the value is in the bounds")
			})
		}
	})

	t.Run("Coerce", func(t *testing.T) {
		t.Parallel()

		narrow := floatBounds(t, 1, 2, choice.ExcludeNaN, choice.Width32)
		tests := []struct {
			name string
			give choice.Choice
		}{
			{name: "returns the target for a value above the range", give: choice.Choice{Kind: choice.Float, Float: 3}},
			{
				name: "returns the target for NaN when the bounds exclude it",
				give: choice.Choice{Kind: choice.Float, Float: math.NaN()},
			},
			{
				name: "returns the target for a value of another width",
				give: choice.Choice{Kind: choice.Float, Float: 1.1},
			},
			{
				name: "returns the target for an integer",
				give: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(1)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, narrow.Coerce(tt.give), 1.0, "the target")
			})
		}

		t.Run("returns a NaN with the canonical bits when the bounds admit it", func(t *testing.T) {
			t.Parallel()
			b := floatBounds(t, 0, 1, choice.AdmitNaN, choice.Width64)
			got := b.Coerce(choice.Choice{Kind: choice.Float, Float: math.NaN()})
			assert.Equal(t, math.Float64bits(got), uint64(choice.NaNBits), "the canonical NaN")
		})

		t.Run("returns -0 recorded inside the range unchanged", func(t *testing.T) {
			t.Parallel()
			b := floatBounds(t, -1, 1, choice.ExcludeNaN, choice.Width64)
			got := b.Coerce(choice.Choice{Kind: choice.Float, Float: negativeZero})
			assert.True(t, choice.SameFloat(got, negativeZero), "the recorded -0")
		})
	})

	t.Run("Integers", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			lo, hi         float64
			w              choice.Width
			wantLo, wantHi int64
			wantOK         bool
		}{
			{
				name:   "returns the integers of a finite range",
				lo:     -2.5,
				hi:     3.5,
				w:      choice.Width64,
				wantLo: -2,
				wantHi: 3,
				wantOK: true,
			},
			{
				name:   "returns one integer for a range around it",
				lo:     0.5,
				hi:     1.5,
				w:      choice.Width64,
				wantLo: 1,
				wantHi: 1,
				wantOK: true,
			},
			{
				name:   "returns 0 for a range around -0",
				lo:     -0.5,
				hi:     0.5,
				w:      choice.Width64,
				wantLo: 0,
				wantHi: 0,
				wantOK: true,
			},
			{
				name:   "returns the integers below 2^53 for every binary64 value",
				lo:     math.Inf(-1),
				hi:     math.Inf(1),
				w:      choice.Width64,
				wantLo: -(1<<53 - 1),
				wantHi: 1<<53 - 1,
				wantOK: true,
			},
			{
				name:   "returns the integers below 2^24 for every binary32 value",
				lo:     math.Inf(-1),
				hi:     math.Inf(1),
				w:      choice.Width32,
				wantLo: -(1<<24 - 1),
				wantHi: 1<<24 - 1,
				wantOK: true,
			},
			{
				name:   "reports false for a range without an integer",
				lo:     0.25,
				hi:     0.75,
				w:      choice.Width64,
				wantOK: false,
			},
			{
				name:   "reports false for a range above 2^53",
				lo:     1 << 60,
				hi:     1 << 61,
				w:      choice.Width64,
				wantOK: false,
			},
			{
				name:   "reports false for positive infinity alone",
				lo:     math.Inf(1),
				hi:     math.Inf(1),
				w:      choice.Width64,
				wantOK: false,
			},
			{
				name:   "reports false for negative infinity alone",
				lo:     math.Inf(-1),
				hi:     math.Inf(-1),
				w:      choice.Width64,
				wantOK: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := floatBounds(t, tt.lo, tt.hi, choice.ExcludeNaN, tt.w).Integers()
				want := [2]choice.Int{choice.IntOf(tt.wantLo), choice.IntOf(tt.wantHi)}
				assert.Equal(t, [2]choice.Int{got.Lo(), got.Hi()}, want, "the integer bounds")
				assert.Equal(t, ok, tt.wantOK, "whether the bounds admit such an integer")
			})
		}
	})
}

// TestFloatBoundsZeroAlloc checks that no method of FloatBounds allocates,
// and that a constructor that succeeds allocates nothing.
func TestFloatBoundsZeroAlloc(t *testing.T) {
	b := floatBounds(t, 0.6, 0.7, choice.AdmitNaN, choice.Width64)
	recorded := choice.Choice{Kind: choice.Float, Float: math.NaN()}
	assert.MaxAllocs(t, func() { _, _ = choice.NewFloatBounds(0.6, 0.7, choice.ExcludeNaN, choice.Width64) }, 0,
		"NewFloatBounds allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Lo() }, 0, "Lo allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Hi() }, 0, "Hi allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.NaNPolicy() }, 0, "NaNPolicy allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Width() }, 0, "Width allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Target() }, 0, "Target allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Admits(0.65) }, 0, "Admits allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Coerce(recorded) }, 0, "Coerce allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = b.Integers() }, 0, "Integers allocates nothing")
}

// BenchmarkFloatBounds measures the constructor and each method of
// FloatBounds under a ceiling of no allocation. The constructor searches
// for the target over the fractional bits of the lower bound.
func BenchmarkFloatBounds(b *testing.B) {
	bounds := floatBounds(b, 0.6, 0.7, choice.AdmitNaN, choice.Width64)

	b.Run("NewFloatBounds", func(b *testing.B) {
		var got choice.FloatBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = choice.NewFloatBounds(0.6, 0.7, choice.AdmitNaN, choice.Width64)
		}
		assert.Equal(b, got.Target(), 0.625, "the simplest value of the range")
	})

	b.Run("Lo", func(b *testing.B) {
		var got float64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Lo()
		}
		assert.Equal(b, got, 0.6, "the lower bound")
	})

	b.Run("Hi", func(b *testing.B) {
		var got float64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Hi()
		}
		assert.Equal(b, got, 0.7, "the upper bound")
	})

	b.Run("NaNPolicy", func(b *testing.B) {
		var got choice.NaNPolicy
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.NaNPolicy()
		}
		assert.Equal(b, got, choice.AdmitNaN, "the NaN policy")
	})

	b.Run("Width", func(b *testing.B) {
		var got choice.Width
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Width()
		}
		assert.Equal(b, got, choice.Width64, "the width")
	})

	b.Run("Target", func(b *testing.B) {
		var got float64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Target()
		}
		assert.Equal(b, got, 0.625, "the simplest value")
	})

	b.Run("Admits", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Admits(0.65)
		}
		assert.True(b, got, "0.65 is in the range")
	})

	b.Run("Coerce", func(b *testing.B) {
		var got float64
		recorded := choice.Choice{Kind: choice.Float, Float: math.NaN()}
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Coerce(recorded)
		}
		assert.Equal(b, math.Float64bits(got), uint64(choice.NaNBits), "the canonical NaN")
	})

	b.Run("Integers", func(b *testing.B) {
		var got choice.IntegerBounds
		whole := floatBounds(b, math.Inf(-1), math.Inf(1), choice.ExcludeNaN, choice.Width64)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = whole.Integers()
		}
		assert.Equal(b, got.Hi(), choice.IntOf(1<<53-1), "the largest exact integer")
	})
}

// floatBounds returns the float bounds [lo, hi] of width w under the NaN
// policy, failing the test when they are invalid.
func floatBounds(tb testing.TB, lo, hi float64, nan choice.NaNPolicy, w choice.Width) choice.FloatBounds {
	tb.Helper()
	b, err := choice.NewFloatBounds(lo, hi, nan, w)
	assert.NoError(tb, err, "the bounds are valid")
	return b
}
