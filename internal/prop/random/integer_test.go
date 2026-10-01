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

// pinnedSeed is the seed of every pinned draw. The pinned draws come from
// the definition's executable reference, so that a change to a draw fails
// here before a vector moves.
const pinnedSeed = 42

// draws is the number of draws that a property of a draw checks: enough
// to reach every branch of every draw.
const draws = 2000

// offsetLimits pins the largest offsets from the target that an integer
// draw allows, one chosen with Below(4).
var offsetLimits = [...]uint64{0xF, 0xFF, 0xFFFF, math.MaxUint64}

// TestInteger checks what an integer draw consumes from the stream, what
// it returns, and the edge values it takes.
func TestInteger(t *testing.T) {
	t.Parallel()

	t.Run("Integer", func(t *testing.T) {
		t.Parallel()

		pinned := []struct {
			name string
			b    choice.IntegerBounds
			want []choice.Int
		}{
			{
				name: "returns the pinned draws of a target inside the bounds",
				b:    signedBounds(t, -100, 100),
				want: ints(11, 24, 14, 62, -71, -5, 8, -21, -29, -69),
			},
			{
				name: "returns the pinned draws over the signed range",
				b:    signedBounds(t, math.MinInt64, math.MaxInt64),
				want: ints(11, 49, 14, 31993, -207, -49792, -10, math.MinInt64, 3534659902462096457, 51818),
			},
			{
				name: "returns the pinned draws upward from a target at the lower bound",
				b:    signedBounds(t, 0, 1000),
				want: ints(51, 352, 13, 14, 8, 12, 822, 778, 43, 2),
			},
			{
				name: "returns the pinned draws downward from a target at the upper bound",
				b:    signedBounds(t, -1000, -1),
				want: ints(-52, -353, -14, -15, -9, -13, -823, -779, -44, -3),
			},
			{
				name: "returns the pinned draws over the unsigned range",
				b:    unsignedBounds(t, 0, math.MaxUint64),
				want: []choice.Int{
					choice.UintOf(51), choice.UintOf(6352690194663110999), choice.UintOf(13), choice.UintOf(14),
					choice.UintOf(8), choice.UintOf(12), choice.UintOf(14808979767600257571), choice.UintOf(49792),
					choice.UintOf(781715023583996500), choice.UintOf(2),
				},
			},
		}
		for _, tt := range pinned {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := random.New(pinnedSeed)
				got := make([]choice.Int, len(tt.want))
				for i := range got {
					got[i] = random.Integer(&s, tt.b)
				}
				assert.Equal(t, got, tt.want, "the first draws of the seed")
			})
		}

		t.Run("returns the value of bounds of one value without consuming the stream", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(1), random.New(1)
			assert.Equal(t, random.Integer(&s, signedBounds(t, 7, 7)), choice.IntOf(7), "the one value")
			assert.Equal(t, s.Next(), twin.Next(), "the next value of the stream")
		})

		t.Run("returns values inside the bounds", func(t *testing.T) {
			t.Parallel()
			for _, b := range []choice.IntegerBounds{
				signedBounds(t, -100, 100),
				signedBounds(t, 0, 1),
				signedBounds(t, 5, 9),
				signedBounds(t, -9, -5),
				signedBounds(t, math.MinInt64, math.MaxInt64),
				unsignedBounds(t, 0, math.MaxUint64),
			} {
				s := random.New(b.Lo().Magnitude())
				for range draws {
					v := random.Integer(&s, b)
					assert.True(t, b.Admits(v), v.String()+" inside ["+b.Lo().String()+", "+b.Hi().String()+"]")
				}
			}
		})

		t.Run("consumes the edge coin then the limit then the offset upward from the lower bound", func(t *testing.T) {
			t.Parallel()
			b := signedBounds(t, 0, 1000)
			for seed := range uint64(50) {
				s, twin := random.New(seed), random.New(seed)
				got := random.Integer(&s, b)
				var want choice.Int
				if twin.Coin(1, 8) {
					edges := random.AppendIntegerEdges(nil, b)
					want = edges[twin.Below(uint64(len(edges)))]
				} else {
					limit := offsetLimits[twin.Below(uint64(len(offsetLimits)))]
					want = choice.UintOf(twin.UpTo(min(1000, limit)))
				}
				assert.Equal(t, got, want, "the draw of seed "+strconv.FormatUint(seed, 10))
				assert.Equal(t, s.Next(), twin.Next(), "the stream after the draw")
			}
		})
	})

	t.Run("AppendIntegerEdges", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			b    choice.IntegerBounds
			want []choice.Int
		}{
			{
				name: "returns the target then the bounds then the neighbours of the target",
				b:    signedBounds(t, -5, 5),
				want: ints(0, -5, 5, 1, -1),
			},
			{name: "returns each value once", b: signedBounds(t, 0, 10), want: ints(0, 10, 1)},
			{
				name: "returns the value below a target at the upper bound",
				b:    signedBounds(t, -9, -3),
				want: ints(-3, -9, -4),
			},
			{name: "returns both values of a range of two", b: signedBounds(t, 1, 2), want: ints(1, 2)},
			{name: "returns the one value of bounds of one value", b: signedBounds(t, 3, 3), want: ints(3)},
			{
				name: "returns the ends of the unsigned range",
				b:    unsignedBounds(t, math.MaxUint64-1, math.MaxUint64),
				want: []choice.Int{choice.UintOf(math.MaxUint64 - 1), choice.UintOf(math.MaxUint64)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, random.AppendIntegerEdges(nil, tt.b), tt.want, "the edge values")
			})
		}

		t.Run("appends after the elements already in dst", func(t *testing.T) {
			t.Parallel()
			got := random.AppendIntegerEdges(ints(0), signedBounds(t, 0, 10))
			assert.Equal(t, got, ints(0, 0, 10, 1), "the elements of dst, then the edge values")
		})
	})
}

// TestIntegerZeroAlloc checks that an integer draw allocates nothing, and
// that AppendIntegerEdges allocates nothing into a slice with the capacity.
func TestIntegerZeroAlloc(t *testing.T) {
	s := random.New(pinnedSeed)
	b := signedBounds(t, math.MinInt64, math.MaxInt64)
	edges := make([]choice.Int, 0, 5)
	assert.MaxAllocs(t, func() { _ = random.Integer(&s, b) }, 0, "Integer allocates nothing")
	assert.MaxAllocs(t, func() { _ = random.AppendIntegerEdges(edges[:0], b) }, 0,
		"AppendIntegerEdges allocates nothing into a slice with the capacity")
}

// BenchmarkInteger measures the integer draw and the edge values under a
// ceiling of no allocation.
func BenchmarkInteger(b *testing.B) {
	bounds := signedBounds(b, math.MinInt64, math.MaxInt64)

	b.Run("Integer", func(b *testing.B) {
		var got choice.Int
		s := random.New(pinnedSeed)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.Integer(&s, bounds)
		}
		assert.True(b, bounds.Admits(got), "a value of the signed range")
	})

	b.Run("AppendIntegerEdges", func(b *testing.B) {
		var got []choice.Int
		edges := make([]choice.Int, 0, 5)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.AppendIntegerEdges(edges[:0], bounds)
		}
		assert.Length(b, got, 5, "the target, both bounds and both neighbours")
	})
}

// signedBounds returns the integer bounds [lo, hi] of two int64 values,
// failing the test when they are invalid.
func signedBounds(tb testing.TB, lo, hi int64) choice.IntegerBounds {
	tb.Helper()
	b, err := choice.NewIntegerBounds(choice.IntOf(lo), choice.IntOf(hi))
	assert.NoError(tb, err, "the bounds are valid")
	return b
}

// unsignedBounds returns the integer bounds [lo, hi] of two uint64 values,
// failing the test when they are invalid.
func unsignedBounds(tb testing.TB, lo, hi uint64) choice.IntegerBounds {
	tb.Helper()
	b, err := choice.NewIntegerBounds(choice.UintOf(lo), choice.UintOf(hi))
	assert.NoError(tb, err, "the bounds are valid")
	return b
}

// ints returns the values as Ints.
func ints(values ...int64) []choice.Int {
	out := make([]choice.Int, len(values))
	for i, v := range values {
		out[i] = choice.IntOf(v)
	}
	return out
}
