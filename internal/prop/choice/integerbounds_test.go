// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"math"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// rankedRanges are small signed bounds whose key order the rank tests
// enumerate: centred, lopsided both ways, one-sided, and one value.
var rankedRanges = [][2]int64{{-5, 10}, {-10, 5}, {3, 9}, {-9, -3}, {-4, 4}, {7, 7}, {-1, 0}}

// TestIntegerBounds checks the construction, the target, the counts on
// each side of it, the admission, the replay and the key order of integer
// bounds, at the ends of both 64-bit ranges.
func TestIntegerBounds(t *testing.T) {
	t.Parallel()

	t.Run("NewIntegerBounds", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name   string
			lo, hi choice.Int
		}{
			{name: "returns the whole signed range", lo: choice.IntOf(math.MinInt64), hi: choice.IntOf(math.MaxInt64)},
			{name: "returns the whole unsigned range", lo: choice.Int{}, hi: choice.UintOf(math.MaxUint64)},
			{name: "returns bounds of one value", lo: choice.IntOf(-7), hi: choice.IntOf(-7)},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				b, err := choice.NewIntegerBounds(tt.lo, tt.hi)
				assert.NoError(t, err, "the bounds are valid")
				assert.Equal(t, [2]choice.Int{b.Lo(), b.Hi()}, [2]choice.Int{tt.lo, tt.hi}, "the bounds as given")
			})
		}

		invalid := []struct {
			name   string
			lo, hi choice.Int
			want   error
		}{
			{
				name: "returns ErrEmpty when lo exceeds hi",
				lo:   choice.IntOf(5),
				hi:   choice.IntOf(4),
				want: choice.ErrEmpty,
			},
			{
				name: "returns ErrRange for -1 to the largest uint64",
				lo:   choice.IntOf(-1),
				hi:   choice.UintOf(math.MaxUint64),
				want: choice.ErrRange,
			},
			{
				name: "returns ErrRange for -1 to 2^63",
				lo:   choice.IntOf(-1),
				hi:   choice.UintOf(minIntMagnitude),
				want: choice.ErrRange,
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := choice.NewIntegerBounds(tt.lo, tt.hi)
				assert.ErrorIs(t, err, tt.want, "the refusal")
			})
		}
	})

	t.Run("MustIntegerBounds", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the bounds that NewIntegerBounds returns", func(t *testing.T) {
			t.Parallel()
			got := choice.MustIntegerBounds(choice.IntOf(-3), choice.IntOf(9))
			assert.Equal(t, got, signedBounds(t, -3, 9), "the bounds")
		})

		t.Run("panics with ErrEmpty when lo exceeds hi", func(t *testing.T) {
			t.Parallel()
			lo, hi := choice.IntOf(5), choice.IntOf(4)
			got := assert.Panics(t, func() { choice.MustIntegerBounds(lo, hi) }, "invalid bounds")
			err, ok := got.(error)
			assert.True(t, ok, "the panic value is the error")
			assert.ErrorIs(t, err, choice.ErrEmpty, "the refusal")
		})
	})

	t.Run("Target", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			lo, hi int64
			want   int64
		}{
			{name: "returns 0 when the bounds straddle it", lo: -5, hi: 5, want: 0},
			{name: "returns 0 when it is the lower bound", lo: 0, hi: 9, want: 0},
			{name: "returns the lower bound above zero", lo: 3, hi: 9, want: 3},
			{name: "returns the upper bound below zero", lo: -9, hi: -3, want: -3},
			{name: "returns the one value of single-value bounds", lo: 7, hi: 7, want: 7},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := signedBounds(t, tt.lo, tt.hi).Target()
				assert.Equal(t, got, choice.IntOf(tt.want), "the value closest to zero")
			})
		}
	})

	t.Run("Above", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.IntegerBounds
			want uint64
		}{
			{name: "returns the count above a target of 0", give: signedBounds(t, -5, 10), want: 10},
			{name: "returns the count above a target at the lower bound", give: signedBounds(t, 3, 9), want: 6},
			{name: "returns 0 for a target at the upper bound", give: signedBounds(t, -9, -3), want: 0},
			{name: "returns 2^64 - 1 for the whole unsigned range", give: unsignedRange(t), want: math.MaxUint64},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Above(), tt.want, "the number of values above the target")
			})
		}
	})

	t.Run("Below", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.IntegerBounds
			want uint64
		}{
			{name: "returns the count below a target of 0", give: signedBounds(t, -5, 10), want: 5},
			{name: "returns the count below a target at the upper bound", give: signedBounds(t, -9, -3), want: 6},
			{name: "returns 0 for a target at the lower bound", give: signedBounds(t, 3, 9), want: 0},
			{name: "returns 2^63 for the whole signed range", give: signedRange(t), want: minIntMagnitude},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Below(), tt.want, "the number of values below the target")
			})
		}
	})

	t.Run("Admits", func(t *testing.T) {
		t.Parallel()

		b := signedBounds(t, -3, 3)
		tests := []struct {
			name string
			give choice.Int
			want bool
		}{
			{name: "reports true for the lower bound", give: choice.IntOf(-3), want: true},
			{name: "reports true for the upper bound", give: choice.IntOf(3), want: true},
			{name: "reports false below the lower bound", give: choice.IntOf(-4), want: false},
			{name: "reports false above the upper bound", give: choice.IntOf(4), want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, b.Admits(tt.give), tt.want, "whether the value is inside the bounds")
			})
		}
	})

	t.Run("Coerce", func(t *testing.T) {
		t.Parallel()

		b := signedBounds(t, 3, 9)
		tests := []struct {
			name string
			give choice.Choice
			want choice.Int
		}{
			{
				name: "returns a recorded value inside the bounds",
				give: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(4)},
				want: choice.IntOf(4),
			},
			{
				name: "returns the target for a value above the bounds",
				give: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(10)},
				want: choice.IntOf(3),
			},
			{
				name: "returns the target for a float",
				give: choice.Choice{Kind: choice.Float, Float: 4},
				want: choice.IntOf(3),
			},
			{
				name: "returns the target for a sequence",
				give: choice.Choice{Kind: choice.Sequence, Sequence: []uint32{4}},
				want: choice.IntOf(3),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, b.Coerce(tt.give), tt.want, "the replayed value")
			})
		}

		t.Run("returns the target for -5 against unsigned bounds", func(t *testing.T) {
			t.Parallel()
			unsigned, err := choice.NewIntegerBounds(choice.Int{}, choice.UintOf(math.MaxUint64))
			assert.NoError(t, err, "the unsigned range is valid")
			recorded := choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(-5)}
			assert.Equal(t, unsigned.Coerce(recorded), choice.Int{}, "a negative value is no unsigned value")
		})
	})

	t.Run("Key", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the zero key for the target", func(t *testing.T) {
			t.Parallel()
			b := signedBounds(t, -9, -3)
			assert.Equal(t, b.Key(b.Target()).Compare(choice.Key{}), 0, "the target is the simplest integer")
		})

		t.Run("orders by distance then above before below", func(t *testing.T) {
			t.Parallel()
			b := signedBounds(t, -2, 2)
			got := []choice.Int{choice.IntOf(-2), choice.IntOf(-1), choice.IntOf(0), choice.IntOf(1), choice.IntOf(2)}
			slices.SortFunc(got, func(x, y choice.Int) int { return b.Key(x).Compare(b.Key(y)) })
			want := []choice.Int{choice.IntOf(0), choice.IntOf(1), choice.IntOf(-1), choice.IntOf(2), choice.IntOf(-2)}
			assert.Equal(t, got, want, "the values in key order")
		})
	})

	t.Run("Rank", func(t *testing.T) {
		t.Parallel()

		t.Run("numbers every value in key order", func(t *testing.T) {
			t.Parallel()
			for _, r := range rankedRanges {
				b := signedBounds(t, r[0], r[1])
				for i, v := range valuesInKeyOrder(b, r[0], r[1]) {
					assert.Equal(t, b.Rank(v), uint64(i), "the rank of "+v.String())
				}
			}
		})

		tests := []struct {
			name string
			b    func(testing.TB) choice.IntegerBounds
			give choice.Int
			want uint64
		}{
			{
				name: "returns the largest uint64 for the largest unsigned value",
				b:    unsignedRange,
				give: choice.UintOf(math.MaxUint64),
				want: math.MaxUint64,
			},
			{
				name: "returns the largest uint64 for the smallest signed value",
				b:    signedRange,
				give: choice.IntOf(math.MinInt64),
				want: math.MaxUint64,
			},
			{
				name: "returns 2^64 - 3 for the largest signed value",
				b:    signedRange,
				give: choice.IntOf(math.MaxInt64),
				want: math.MaxUint64 - 2,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.b(t).Rank(tt.give), tt.want, "the rank at the end of the range")
			})
		}
	})

	t.Run("AtRank", func(t *testing.T) {
		t.Parallel()

		t.Run("returns every value in key order", func(t *testing.T) {
			t.Parallel()
			for _, r := range rankedRanges {
				b := signedBounds(t, r[0], r[1])
				for i, want := range valuesInKeyOrder(b, r[0], r[1]) {
					assert.Equal(t, b.AtRank(uint64(i)), want, "the value at its rank")
				}
			}
		})

		t.Run("continues on the longer side past the shorter one", func(t *testing.T) {
			t.Parallel()
			b := signedBounds(t, -2, 9)
			got := []choice.Int{b.AtRank(4), b.AtRank(5), b.AtRank(6), b.AtRank(7)}
			want := []choice.Int{choice.IntOf(-2), choice.IntOf(3), choice.IntOf(4), choice.IntOf(5)}
			assert.Equal(t, got, want, "-2, then one value per rank above")
		})

		tests := []struct {
			name string
			b    func(testing.TB) choice.IntegerBounds
			give uint64
			want choice.Int
		}{
			{
				name: "returns the largest unsigned value at the last rank",
				b:    unsignedRange,
				give: math.MaxUint64,
				want: choice.UintOf(math.MaxUint64),
			},
			{
				name: "returns the smallest signed value at the last rank",
				b:    signedRange,
				give: math.MaxUint64,
				want: choice.IntOf(math.MinInt64),
			},
			{
				name: "returns the largest signed value at rank 2^64 - 3",
				b:    signedRange,
				give: math.MaxUint64 - 2,
				want: choice.IntOf(math.MaxInt64),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.b(t).AtRank(tt.give), tt.want, "the value at the end of the key order")
			})
		}
	})
}

// TestIntegerBoundsAllocs checks that no method of IntegerBounds
// allocates, and that a constructor that succeeds allocates nothing.
func TestIntegerBoundsAllocs(t *testing.T) {
	lo, hi := choice.IntOf(-9), choice.IntOf(9)
	b := signedBounds(t, -9, 9)
	recorded := choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(4)}
	assert.MaxAllocs(t, func() { _, _ = choice.NewIntegerBounds(lo, hi) }, 0, "NewIntegerBounds allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.MustIntegerBounds(lo, hi) }, 0, "MustIntegerBounds allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Lo() }, 0, "Lo allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Hi() }, 0, "Hi allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Target() }, 0, "Target allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Above() }, 0, "Above allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Below() }, 0, "Below allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Admits(hi) }, 0, "Admits allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Coerce(recorded) }, 0, "Coerce allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Key(lo) }, 0, "Key allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Rank(lo) }, 0, "Rank allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.AtRank(7) }, 0, "AtRank allocates nothing")
}

// BenchmarkIntegerBounds measures the constructor and each method of
// IntegerBounds under a ceiling of no allocation.
func BenchmarkIntegerBounds(b *testing.B) {
	lo, hi := choice.IntOf(math.MinInt64), choice.IntOf(math.MaxInt64)
	bounds := signedRange(b)

	b.Run("NewIntegerBounds", func(b *testing.B) {
		var got choice.IntegerBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = choice.NewIntegerBounds(lo, hi)
		}
		assert.Equal(b, got, bounds, "the whole signed range")
	})

	b.Run("MustIntegerBounds", func(b *testing.B) {
		var got choice.IntegerBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.MustIntegerBounds(lo, hi)
		}
		assert.Equal(b, got, bounds, "the whole signed range")
	})

	b.Run("Lo", func(b *testing.B) {
		var got choice.Int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Lo()
		}
		assert.Equal(b, got, lo, "the lower bound")
	})

	b.Run("Hi", func(b *testing.B) {
		var got choice.Int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Hi()
		}
		assert.Equal(b, got, hi, "the upper bound")
	})

	b.Run("Target", func(b *testing.B) {
		var got choice.Int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Target()
		}
		assert.Equal(b, got, choice.Int{}, "zero")
	})

	b.Run("Above", func(b *testing.B) {
		var got uint64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Above()
		}
		assert.Equal(b, got, uint64(math.MaxInt64), "the values above zero")
	})

	b.Run("Below", func(b *testing.B) {
		var got uint64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Below()
		}
		assert.Equal(b, got, uint64(minIntMagnitude), "the values below zero")
	})

	b.Run("Admits", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Admits(lo)
		}
		assert.True(b, got, "the lower bound is admitted")
	})

	b.Run("Coerce", func(b *testing.B) {
		var got choice.Int
		recorded := choice.Choice{Kind: choice.Integer, Integer: lo}
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Coerce(recorded)
		}
		assert.Equal(b, got, lo, "the recorded value")
	})

	b.Run("Key", func(b *testing.B) {
		var got choice.Key
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Key(lo)
		}
		assert.Equal(b, got.Compare(bounds.Key(hi)), 1, "the lower bound is farther from zero")
	})

	b.Run("Rank", func(b *testing.B) {
		var got uint64
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Rank(lo)
		}
		assert.Equal(b, got, uint64(math.MaxUint64), "the last rank")
	})

	b.Run("AtRank", func(b *testing.B) {
		var got choice.Int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.AtRank(math.MaxUint64)
		}
		assert.Equal(b, got, lo, "the value at the last rank")
	})
}

// signedRange returns the bounds of the whole signed 64-bit range.
func signedRange(tb testing.TB) choice.IntegerBounds {
	tb.Helper()
	return signedBounds(tb, math.MinInt64, math.MaxInt64)
}

// unsignedRange returns the bounds of the whole unsigned 64-bit range.
func unsignedRange(tb testing.TB) choice.IntegerBounds {
	tb.Helper()
	b, err := choice.NewIntegerBounds(choice.Int{}, choice.UintOf(math.MaxUint64))
	assert.NoError(tb, err, "the unsigned range is valid")
	return b
}

// valuesInKeyOrder returns every value of the bounds [lo, hi], sorted by
// the bounds' keys, which the rank functions must follow.
func valuesInKeyOrder(b choice.IntegerBounds, lo, hi int64) []choice.Int {
	values := make([]choice.Int, 0, hi-lo+1)
	for v := lo; v <= hi; v++ {
		values = append(values, choice.IntOf(v))
	}
	slices.SortFunc(values, func(x, y choice.Int) int { return b.Key(x).Compare(b.Key(y)) })
	return values
}
