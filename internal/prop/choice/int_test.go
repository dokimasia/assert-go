// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestInt checks the conversions into and out of Int, its order, its
// arithmetic and its text, at the ends of both 64-bit ranges.
func TestInt(t *testing.T) {
	t.Parallel()

	t.Run("IntOf", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name          string
			give          int64
			wantMagnitude uint64
			wantNegative  bool
		}{
			{
				name:          "returns a magnitude of 2^63 for the smallest int64",
				give:          math.MinInt64,
				wantMagnitude: minIntMagnitude,
				wantNegative:  true,
			},
			{name: "returns a negative magnitude of 1 for -1", give: -1, wantMagnitude: 1, wantNegative: true},
			{name: "returns the zero value for 0", give: 0, wantMagnitude: 0, wantNegative: false},
			{
				name:          "returns the largest int64 as its magnitude",
				give:          math.MaxInt64,
				wantMagnitude: math.MaxInt64,
				wantNegative:  false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := choice.IntOf(tt.give)
				assert.Equal(t, got.Magnitude(), tt.wantMagnitude, "the magnitude")
				assert.Equal(t, got.Negative(), tt.wantNegative, "the sign")
			})
		}

		t.Run("returns a value equal to the zero Int for 0", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, choice.IntOf(0), choice.Int{}, "zero has one representation")
		})
	})

	t.Run("UintOf", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the largest uint64 as a positive magnitude", func(t *testing.T) {
			t.Parallel()
			got := choice.UintOf(math.MaxUint64)
			assert.Equal(t, got.Magnitude(), uint64(math.MaxUint64), "the magnitude")
			assert.False(t, got.Negative(), "the largest uint64 is positive")
		})

		t.Run("returns the same value as IntOf below 2^63", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, choice.UintOf(7), choice.IntOf(7), "one value, one representation")
		})
	})

	t.Run("Int64", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			give   choice.Int
			want   int64
			wantOK bool
		}{
			{name: "returns the smallest int64", give: choice.IntOf(math.MinInt64), want: math.MinInt64, wantOK: true},
			{name: "returns -1", give: choice.IntOf(-1), want: -1, wantOK: true},
			{name: "returns the largest int64", give: choice.IntOf(math.MaxInt64), want: math.MaxInt64, wantOK: true},
			{
				name:   "reports false above the largest int64",
				give:   choice.UintOf(minIntMagnitude),
				want:   0,
				wantOK: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := tt.give.Int64()
				assert.Equal(t, got, tt.want, "the int64")
				assert.Equal(t, ok, tt.wantOK, "whether the value fits")
			})
		}
	})

	t.Run("Uint64", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			give   choice.Int
			want   uint64
			wantOK bool
		}{
			{
				name:   "returns the largest uint64",
				give:   choice.UintOf(math.MaxUint64),
				want:   math.MaxUint64,
				wantOK: true,
			},
			{name: "returns 0", give: choice.Int{}, want: 0, wantOK: true},
			{name: "reports false for -1", give: choice.IntOf(-1), want: 0, wantOK: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := tt.give.Uint64()
				assert.Equal(t, got, tt.want, "the uint64")
				assert.Equal(t, ok, tt.wantOK, "whether the value fits")
			})
		}
	})

	t.Run("Float64", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Int
			want float64
		}{
			{name: "returns -2^63 for the smallest int64", give: choice.IntOf(math.MinInt64), want: -minIntMagnitude},
			{name: "returns -3 for -3", give: choice.IntOf(-3), want: -3},
			{name: "returns 0 for the zero value", give: choice.Int{}, want: 0},
			{name: "returns 2^53 - 1 exactly", give: choice.UintOf(1<<53 - 1), want: 1<<53 - 1},
			{name: "returns 2^64 for the largest uint64", give: choice.UintOf(math.MaxUint64), want: 1 << 64},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Float64(), tt.want, "the nearest float64")
			})
		}
	})

	t.Run("Compare", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			i, j choice.Int
			want int
		}{
			{name: "returns -1 for -2 against -1", i: choice.IntOf(-2), j: choice.IntOf(-1), want: -1},
			{name: "returns 1 for -1 against -2", i: choice.IntOf(-1), j: choice.IntOf(-2), want: 1},
			{name: "returns -1 for -1 against 0", i: choice.IntOf(-1), j: choice.IntOf(0), want: -1},
			{name: "returns 1 for 0 against -1", i: choice.IntOf(0), j: choice.IntOf(-1), want: 1},
			{name: "returns 0 for -3 against -3", i: choice.IntOf(-3), j: choice.IntOf(-3), want: 0},
			{name: "returns 0 for 3 against 3", i: choice.IntOf(3), j: choice.UintOf(3), want: 0},
			{
				name: "returns -1 for 1 against the largest uint64",
				i:    choice.IntOf(1),
				j:    choice.UintOf(math.MaxUint64),
				want: -1,
			},
			{
				name: "returns 1 for the largest uint64 against 1",
				i:    choice.UintOf(math.MaxUint64),
				j:    choice.IntOf(1),
				want: 1,
			},
			{
				name: "returns -1 for the smallest int64 against the largest uint64",
				i:    choice.IntOf(math.MinInt64),
				j:    choice.UintOf(math.MaxUint64),
				want: -1,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.i.Compare(tt.j), tt.want, "the order of the two values")
			})
		}
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Int
			d    uint64
			want choice.Int
		}{
			{name: "returns 8 for 5 plus 3", give: choice.IntOf(5), d: 3, want: choice.IntOf(8)},
			{name: "returns -2 for -5 plus 3", give: choice.IntOf(-5), d: 3, want: choice.IntOf(-2)},
			{name: "returns the zero value for -5 plus 5", give: choice.IntOf(-5), d: 5, want: choice.Int{}},
			{name: "returns 3 for -5 plus 8", give: choice.IntOf(-5), d: 8, want: choice.IntOf(3)},
			{
				name: "returns the largest uint64 for 0 plus it",
				give: choice.Int{},
				d:    math.MaxUint64,
				want: choice.UintOf(math.MaxUint64),
			},
			{
				name: "returns the largest int64 for the smallest int64 plus 2^64 - 1",
				give: choice.IntOf(math.MinInt64),
				d:    math.MaxUint64,
				want: choice.IntOf(math.MaxInt64),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Add(tt.d), tt.want, "the sum")
			})
		}
	})

	t.Run("Sub", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Int
			d    uint64
			want choice.Int
		}{
			{name: "returns 2 for 5 minus 3", give: choice.IntOf(5), d: 3, want: choice.IntOf(2)},
			{name: "returns the zero value for 5 minus 5", give: choice.IntOf(5), d: 5, want: choice.Int{}},
			{name: "returns -3 for 5 minus 8", give: choice.IntOf(5), d: 8, want: choice.IntOf(-3)},
			{name: "returns -8 for -5 minus 3", give: choice.IntOf(-5), d: 3, want: choice.IntOf(-8)},
			{
				name: "returns the smallest int64 for 0 minus 2^63",
				give: choice.Int{},
				d:    minIntMagnitude,
				want: choice.IntOf(math.MinInt64),
			},
			{
				name: "returns the smallest int64 for the largest int64 minus 2^64 - 1",
				give: choice.IntOf(math.MaxInt64),
				d:    math.MaxUint64,
				want: choice.IntOf(math.MinInt64),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Sub(tt.d), tt.want, "the difference")
			})
		}
	})

	t.Run("Distance", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			i, j choice.Int
			want uint64
		}{
			{name: "returns 3 for 5 and 2", i: choice.IntOf(5), j: choice.IntOf(2), want: 3},
			{name: "returns 3 for 2 and 5", i: choice.IntOf(2), j: choice.IntOf(5), want: 3},
			{name: "returns 7 for -5 and 2", i: choice.IntOf(-5), j: choice.IntOf(2), want: 7},
			{name: "returns 3 for -5 and -2", i: choice.IntOf(-5), j: choice.IntOf(-2), want: 3},
			{
				name: "returns 2^64 - 1 for the ends of the signed range",
				i:    choice.IntOf(math.MinInt64),
				j:    choice.IntOf(math.MaxInt64),
				want: math.MaxUint64,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.i.Distance(tt.j), tt.want, "the distance")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Int
			want string
		}{
			{
				name: "returns the smallest int64 with a minus sign",
				give: choice.IntOf(math.MinInt64),
				want: "-9223372036854775808",
			},
			{name: "returns 0 for the zero value", give: choice.Int{}, want: "0"},
			{
				name: "returns the largest uint64 in decimal",
				give: choice.UintOf(math.MaxUint64),
				want: "18446744073709551615",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the decimal text")
			})
		}
	})
}

// TestIntAllocs checks that every function and method of Int allocates
// nothing, but String, which allocates its text.
func TestIntAllocs(t *testing.T) {
	negative, large := choice.IntOf(-5), choice.UintOf(math.MaxUint64)
	assert.MaxAllocs(t, func() { _ = large.String() }, 1, "String allocates the text it returns")
	assert.MaxAllocs(t, func() { _ = choice.IntOf(7).String() }, 0, "String of a single digit allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.IntOf(-5) }, 0, "IntOf allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.UintOf(5) }, 0, "UintOf allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = negative.Int64() }, 0, "Int64 allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = large.Uint64() }, 0, "Uint64 allocates nothing")
	assert.MaxAllocs(t, func() { _ = negative.Float64() }, 0, "Float64 allocates nothing")
	assert.MaxAllocs(t, func() { _ = negative.Negative() }, 0, "Negative allocates nothing")
	assert.MaxAllocs(t, func() { _ = negative.Magnitude() }, 0, "Magnitude allocates nothing")
	assert.MaxAllocs(t, func() { _ = negative.Compare(large) }, 0, "Compare allocates nothing")
	assert.MaxAllocs(t, func() { _ = negative.Add(9) }, 0, "Add allocates nothing")
	assert.MaxAllocs(t, func() { _ = negative.Sub(9) }, 0, "Sub allocates nothing")
	assert.MaxAllocs(t, func() { _ = negative.Distance(large) }, 0, "Distance allocates nothing")
}

// BenchmarkInt measures each function and method of Int. String allocates
// the text it returns.
func BenchmarkInt(b *testing.B) {
	b.Run("IntOf", func(b *testing.B) {
		var got choice.Int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.IntOf(math.MinInt64)
		}
		assert.Equal(b, got.Magnitude(), uint64(minIntMagnitude), "the magnitude")
	})

	b.Run("UintOf", func(b *testing.B) {
		var got choice.Int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.UintOf(math.MaxUint64)
		}
		assert.Equal(b, got.Magnitude(), uint64(math.MaxUint64), "the magnitude")
	})

	b.Run("Int64", func(b *testing.B) {
		var got int64
		i := choice.IntOf(math.MinInt64)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = i.Int64()
		}
		assert.Equal(b, got, int64(math.MinInt64), "the int64")
	})

	b.Run("Uint64", func(b *testing.B) {
		var got uint64
		i := choice.UintOf(math.MaxUint64)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = i.Uint64()
		}
		assert.Equal(b, got, uint64(math.MaxUint64), "the uint64")
	})

	b.Run("Float64", func(b *testing.B) {
		var got float64
		i := choice.IntOf(-3)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = i.Float64()
		}
		assert.Equal(b, got, -3.0, "the float64")
	})

	b.Run("Negative", func(b *testing.B) {
		var got bool
		i := choice.IntOf(-1)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = i.Negative()
		}
		assert.True(b, got, "-1 is negative")
	})

	b.Run("Magnitude", func(b *testing.B) {
		var got uint64
		i := choice.IntOf(-1)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = i.Magnitude()
		}
		assert.Equal(b, got, uint64(1), "the magnitude")
	})

	b.Run("Compare", func(b *testing.B) {
		var got int
		i, j := choice.IntOf(-2), choice.IntOf(-1)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = i.Compare(j)
		}
		assert.Equal(b, got, -1, "-2 is less than -1")
	})

	b.Run("Add", func(b *testing.B) {
		var got choice.Int
		i := choice.IntOf(-5)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = i.Add(8)
		}
		assert.Equal(b, got, choice.IntOf(3), "the sum")
	})

	b.Run("Sub", func(b *testing.B) {
		var got choice.Int
		i := choice.IntOf(5)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = i.Sub(8)
		}
		assert.Equal(b, got, choice.IntOf(-3), "the difference")
	})

	b.Run("Distance", func(b *testing.B) {
		var got uint64
		i, j := choice.IntOf(-5), choice.IntOf(2)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = i.Distance(j)
		}
		assert.Equal(b, got, uint64(7), "the distance")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		i := choice.IntOf(math.MinInt64)
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got = i.String()
		}
		assert.Equal(b, got, "-9223372036854775808", "the decimal text")
	})
}
