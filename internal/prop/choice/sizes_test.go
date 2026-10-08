// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestSizes checks the construction of sizes, the lengths they admit, the
// clamping of a length, and the bounds of a collection's continue flag.
func TestSizes(t *testing.T) {
	t.Parallel()

	t.Run("NewSizes", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name             string
			minSize, maxSize int
		}{
			{name: "returns ErrEmpty for a negative minimum", minSize: -1, maxSize: 1},
			{name: "returns ErrEmpty for a maximum below the minimum", minSize: 3, maxSize: 2},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := choice.NewSizes(tt.minSize, tt.maxSize)
				assert.ErrorIs(t, err, choice.ErrEmpty, "the refusal")
			})
		}

		t.Run("returns sizes of one length when the minimum is the maximum", func(t *testing.T) {
			t.Parallel()
			s := sizes(t, 3, 3)
			maxSize, bounded := s.Max()
			assert.Equal(t, [2]int{s.Min(), maxSize}, [2]int{3, 3}, "the minimum and the maximum")
			assert.True(t, bounded, "the length has a maximum")
		})
	})

	t.Run("NewUnboundedSizes", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrEmpty for a negative minimum", func(t *testing.T) {
			t.Parallel()
			_, err := choice.NewUnboundedSizes(-1)
			assert.ErrorIs(t, err, choice.ErrEmpty, "the refusal")
		})

		t.Run("returns sizes without a maximum", func(t *testing.T) {
			t.Parallel()
			s := unboundedSizes(t, 2)
			_, bounded := s.Max()
			assert.Equal(t, s.Min(), 2, "the minimum")
			assert.False(t, bounded, "the length has no maximum")
		})
	})

	t.Run("Admits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Sizes
			n    int
			want bool
		}{
			{name: "reports true for the minimum", give: sizes(t, 2, 4), n: 2, want: true},
			{name: "reports true for the maximum", give: sizes(t, 2, 4), n: 4, want: true},
			{name: "reports false below the minimum", give: sizes(t, 2, 4), n: 1, want: false},
			{name: "reports false above the maximum", give: sizes(t, 2, 4), n: 5, want: false},
			{
				name: "reports true for a long length without a maximum",
				give: unboundedSizes(t, 2),
				n:    1 << 20,
				want: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Admits(tt.n), tt.want, "whether the sizes admit the length")
			})
		}
	})

	t.Run("Clamp", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Sizes
			n    int
			want int
		}{
			{name: "returns the minimum for a length below it", give: sizes(t, 2, 4), n: 0, want: 2},
			{name: "returns the maximum for a length above it", give: sizes(t, 2, 4), n: 9, want: 4},
			{name: "returns a length the sizes admit", give: sizes(t, 2, 4), n: 3, want: 3},
			{name: "returns a long length without a maximum", give: unboundedSizes(t, 2), n: 9, want: 9},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Clamp(tt.n), tt.want, "the nearest length")
			})
		}
	})

	t.Run("FlagBounds", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			give   choice.Sizes
			count  int
			lo, hi uint64
		}{
			{name: "returns [1, 1] below the minimum", give: sizes(t, 2, 4), count: 1, lo: 1, hi: 1},
			{name: "returns [0, 1] at the minimum", give: sizes(t, 2, 4), count: 2, lo: 0, hi: 1},
			{name: "returns [0, 1] below the maximum", give: sizes(t, 2, 4), count: 3, lo: 0, hi: 1},
			{name: "returns [0, 0] at the maximum", give: sizes(t, 2, 4), count: 4, lo: 0, hi: 0},
			{
				name:  "returns [0, 1] for any count without a maximum",
				give:  unboundedSizes(t, 0),
				count: 100,
				lo:    0,
				hi:    1,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				b := tt.give.FlagBounds(tt.count)
				want := [2]choice.Int{choice.UintOf(tt.lo), choice.UintOf(tt.hi)}
				assert.Equal(t, [2]choice.Int{b.Lo(), b.Hi()}, want, "the bounds of the flag")
			})
		}
	})
}

// TestSizesAllocs checks that the constructors and every method of
// Sizes allocate nothing.
func TestSizesAllocs(t *testing.T) {
	s := sizes(t, 2, 4)
	assert.MaxAllocs(t, func() { _, _ = choice.NewSizes(2, 4) }, 0, "NewSizes allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = choice.NewUnboundedSizes(2) }, 0, "NewUnboundedSizes allocates nothing")
	assert.MaxAllocs(t, func() { _ = s.Min() }, 0, "Min allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = s.Max() }, 0, "Max allocates nothing")
	assert.MaxAllocs(t, func() { _ = s.Admits(3) }, 0, "Admits allocates nothing")
	assert.MaxAllocs(t, func() { _ = s.Clamp(9) }, 0, "Clamp allocates nothing")
	assert.MaxAllocs(t, func() { _ = s.FlagBounds(3) }, 0, "FlagBounds allocates nothing")
}

// BenchmarkSizes measures the constructors and each method of Sizes under
// a ceiling of no allocation.
func BenchmarkSizes(b *testing.B) {
	s := sizes(b, 2, 4)

	b.Run("NewSizes", func(b *testing.B) {
		var got choice.Sizes
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = choice.NewSizes(2, 4)
		}
		assert.Equal(b, got, s, "the sizes")
	})

	b.Run("NewUnboundedSizes", func(b *testing.B) {
		var got choice.Sizes
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = choice.NewUnboundedSizes(2)
		}
		assert.Equal(b, got.Min(), 2, "the minimum")
	})

	b.Run("Min", func(b *testing.B) {
		var got int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Min()
		}
		assert.Equal(b, got, 2, "the minimum")
	})

	b.Run("Max", func(b *testing.B) {
		var got int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = s.Max()
		}
		assert.Equal(b, got, 4, "the maximum")
	})

	b.Run("Admits", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Admits(3)
		}
		assert.True(b, got, "3 is a length of the sizes")
	})

	b.Run("Clamp", func(b *testing.B) {
		var got int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Clamp(9)
		}
		assert.Equal(b, got, 4, "the maximum")
	})

	b.Run("FlagBounds", func(b *testing.B) {
		var got choice.IntegerBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.FlagBounds(3)
		}
		assert.Equal(b, got.Hi(), choice.UintOf(1), "a free decision")
	})
}
