// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"math"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
)

// Seeds whose first float draw takes one branch, found with the
// definition's executable reference.
const (
	// nanSeed is a seed whose first float draw at width 32 assembles a NaN
	// from its bits.
	nanSeed = 87
	// lastIntegralSeed is a seed whose first float draw takes branch 3, the
	// last integral branch, and returns 7 over [-10, 10].
	lastIntegralSeed = 6
)

// The pinned float draws of pinnedSeed, as the bits of each float64.
var (
	// pinnedEveryBinary64 are the first 16 draws over every binary64
	// value: assembled bits and integral values.
	pinnedEveryBinary64 = []uint64{
		0x19FB3C7F3CD32908, 0x27C3103EE57275F0, 0x024EDB73CB92F822, 0xC040800000000000,
		0x67BCD8410A325C29, 0xE14AFCD3D608DF87, 0x53D0146C6810E482, 0x432886D13A1808FA,
		0xC31DFB0215270A70, 0xD398AAC05765F374, 0x007B5F27081A8582, 0xAD40C0B6D33F8431,
		0xC04C800000000000, 0xDB393C4DC33B2715, 0x23BD4C2AE04E5136, 0x11892F614210B0DB,
	}
	// pinnedBinary32 are the first 16 draws over [-10, 10] at width 32,
	// among them assembled values that the bounds cut to -10 and to 10.
	pinnedBinary32 = []uint64{
		0x3B3B3C7F20000000, 0x3CF3103EE0000000, 0x384EDB73C0000000, 0xC000000000000000,
		0x4024000000000000, 0xC024000000000000, 0x4024000000000000, 0x4018000000000000,
		0xC008000000000000, 0xC024000000000000, 0x3806BE4E00000000, 0xBDA0C0B6C0000000,
		0x0000000000000000, 0xC024000000000000, 0x3C7D4C2AE0000000, 0x3A392F6140000000,
	}
	// pinnedWithoutIntegers are the first 16 draws over [0.5, 0.75], which
	// contain no integer: edge values, and assembled values cut to a bound.
	pinnedWithoutIntegers = []float64{
		0.5, 0.5, 0.5, 0.75, 0.5, 0.5, 0.5, 0.5, 0.75, 0.75, 0.5, 0.5, 0.5, 0.75, 0.5, 0.5,
	}
)

// TestFloat checks what a float draw returns from each branch, and the
// edge values it takes.
func TestFloat(t *testing.T) {
	t.Parallel()

	t.Run("Float", func(t *testing.T) {
		t.Parallel()

		pinned := []struct {
			name string
			b    choice.FloatBounds
			want []uint64
		}{
			{
				name: "returns the pinned draws over every binary64 value",
				b:    floatBounds(t, math.Inf(-1), math.Inf(1), choice.ExcludeNaN, choice.Width64),
				want: pinnedEveryBinary64,
			},
			{
				name: "returns the pinned draws over a binary32 range",
				b:    floatBounds(t, -10, 10, choice.ExcludeNaN, choice.Width32),
				want: pinnedBinary32,
			},
			{
				name: "returns the pinned draws over a range without an integer",
				b:    floatBounds(t, 0.5, 0.75, choice.ExcludeNaN, choice.Width64),
				want: floatBits(pinnedWithoutIntegers),
			},
		}
		for _, tt := range pinned {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := random.New(pinnedSeed)
				got := make([]float64, len(tt.want))
				for i := range got {
					got[i] = random.Float(&s, tt.b)
				}
				assert.Equal(t, floatBits(got), tt.want, "the first draws of the seed")
			})
		}

		t.Run("returns an integral value from the last integral branch", func(t *testing.T) {
			t.Parallel()
			s := random.New(lastIntegralSeed)
			got := random.Float(&s, floatBounds(t, -10, 10, choice.ExcludeNaN, choice.Width64))
			assert.Equal(t, got, 7.0, "the integral draw of branch 3")
		})

		t.Run("returns the canonical NaN for assembled NaN bits when the bounds admit NaN", func(t *testing.T) {
			t.Parallel()
			s := random.New(nanSeed)
			got := random.Float(&s, floatBounds(t, math.Inf(-1), math.Inf(1), choice.AdmitNaN, choice.Width32))
			assert.Equal(t, math.Float64bits(got), uint64(choice.NaNBits), "the canonical NaN")
		})

		t.Run("returns the target for assembled NaN bits when the bounds exclude NaN", func(t *testing.T) {
			t.Parallel()
			s := random.New(nanSeed)
			got := random.Float(&s, floatBounds(t, -1, 1, choice.ExcludeNaN, choice.Width32))
			assert.Equal(t, math.Float64bits(got), uint64(0), "the target, +0")
		})

		t.Run("returns values the bounds admit", func(t *testing.T) {
			t.Parallel()
			for _, b := range []choice.FloatBounds{
				floatBounds(t, -1, 1, choice.ExcludeNaN, choice.Width64),
				floatBounds(t, 0.5, 0.75, choice.ExcludeNaN, choice.Width64),
				floatBounds(t, math.Inf(-1), math.Inf(1), choice.ExcludeNaN, choice.Width64),
				floatBounds(t, 1e300, math.Inf(1), choice.ExcludeNaN, choice.Width64),
				floatBounds(t, math.Inf(1), math.Inf(1), choice.ExcludeNaN, choice.Width64),
				floatBounds(t, math.Inf(-1), math.Inf(-1), choice.ExcludeNaN, choice.Width64),
				floatBounds(t, -10, 10, choice.ExcludeNaN, choice.Width32),
				floatBounds(t, math.Inf(-1), math.Inf(1), choice.AdmitNaN, choice.Width32),
			} {
				s := random.New(17)
				for range draws {
					v := random.Float(&s, b)
					assert.True(t, b.Admits(v), strconv.FormatFloat(v, 'g', -1, 64)+" admitted by the bounds")
				}
			}
		})

		t.Run("returns NaN only from bounds that admit it", func(t *testing.T) {
			t.Parallel()
			s := random.New(23)
			admitted := floatBounds(t, math.Inf(-1), math.Inf(1), choice.AdmitNaN, choice.Width64)
			excluded := floatBounds(t, math.Inf(-1), math.Inf(1), choice.ExcludeNaN, choice.Width64)
			admittedNaNs, excludedNaNs := 0, 0
			for range draws {
				if math.IsNaN(random.Float(&s, admitted)) {
					admittedNaNs++
				}
				if math.IsNaN(random.Float(&s, excluded)) {
					excludedNaNs++
				}
			}
			assert.True(t, admittedNaNs > 0, "bounds that admit NaN produce it")
			assert.Equal(t, excludedNaNs, 0, "bounds that exclude NaN never produce it")
		})
	})

	t.Run("AppendFloatEdges", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			b    choice.FloatBounds
			want []float64
		}{
			{
				name: "returns the target then the bounds then the neighbours of the target then NaN",
				b:    floatBounds(t, -1, 1, choice.AdmitNaN, choice.Width64),
				want: []float64{0, -1, 1, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, choice.NaN()},
			},
			{
				name: "returns the neighbours of the target at width 32",
				b:    floatBounds(t, -1, 1, choice.ExcludeNaN, choice.Width32),
				want: []float64{0, -1, 1, math.SmallestNonzeroFloat32, -math.SmallestNonzeroFloat32},
			},
			{
				name: "returns each value once",
				b:    floatBounds(t, 0.5, 0.75, choice.ExcludeNaN, choice.Width64),
				want: []float64{0.5, 0.75, math.Nextafter(0.5, 1)},
			},
			{
				name: "returns positive infinity alone for bounds of it",
				b:    floatBounds(t, math.Inf(1), math.Inf(1), choice.ExcludeNaN, choice.Width64),
				want: []float64{math.Inf(1)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := random.AppendFloatEdges(nil, tt.b)
				assert.Equal(t, floatBits(got), floatBits(tt.want), "the edge values")
			})
		}

		t.Run("appends after the elements already in dst", func(t *testing.T) {
			t.Parallel()
			got := random.AppendFloatEdges([]float64{0.5}, floatBounds(t, 0.5, 0.75, choice.ExcludeNaN, choice.Width64))
			want := []float64{0.5, 0.5, 0.75, math.Nextafter(0.5, 1)}
			assert.Equal(t, floatBits(got), floatBits(want), "the elements of dst, then the edge values")
		})
	})
}

// TestFloatZeroAlloc checks that a float draw allocates nothing, and that
// AppendFloatEdges allocates nothing into a slice with the capacity.
func TestFloatZeroAlloc(t *testing.T) {
	s := random.New(pinnedSeed)
	b := floatBounds(t, math.Inf(-1), math.Inf(1), choice.AdmitNaN, choice.Width64)
	edges := make([]float64, 0, 6)
	assert.MaxAllocs(t, func() { _ = random.Float(&s, b) }, 0, "Float allocates nothing")
	assert.MaxAllocs(t, func() { _ = random.AppendFloatEdges(edges[:0], b) }, 0,
		"AppendFloatEdges allocates nothing into a slice with the capacity")
}

// BenchmarkFloat measures the float draw and the edge values under a
// ceiling of no allocation.
func BenchmarkFloat(b *testing.B) {
	bounds := floatBounds(b, math.Inf(-1), math.Inf(1), choice.AdmitNaN, choice.Width64)

	b.Run("Float", func(b *testing.B) {
		var got float64
		s := random.New(pinnedSeed)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.Float(&s, bounds)
		}
		assert.True(b, bounds.Admits(got), "a value of the bounds")
	})

	b.Run("AppendFloatEdges", func(b *testing.B) {
		var got []float64
		edges := make([]float64, 0, 6)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.AppendFloatEdges(edges[:0], bounds)
		}
		assert.Length(b, got, 6, "the target, both bounds, both neighbours and NaN")
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

// floatBits returns the bits of each value, so that a comparison tells -0
// from +0 and compares NaNs by their bits.
func floatBits(values []float64) []uint64 {
	out := make([]uint64, len(values))
	for i, v := range values {
		out[i] = math.Float64bits(v)
	}
	return out
}
