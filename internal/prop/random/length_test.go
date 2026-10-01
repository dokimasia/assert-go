// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
)

// TestLength checks the average length of a collection and the decisions
// that build its length one element at a time.
func TestLength(t *testing.T) {
	t.Parallel()

	t.Run("Average", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Sizes
			want int
		}{
			{name: "returns 5 for an unbounded length", give: unboundedSizes(t, 0), want: 5},
			{name: "returns 5 for a maximum far above the minimum", give: sizes(t, 0, 16), want: 5},
			{name: "returns half the range rounded up for a small range", give: sizes(t, 0, 3), want: 2},
			{name: "returns 1 for a maximum of 1", give: sizes(t, 0, 1), want: 1},
			{name: "returns the minimum plus half the range above it", give: sizes(t, 2, 4), want: 3},
			{name: "returns the one length of a fixed size", give: sizes(t, 3, 3), want: 3},
			{name: "returns twice a minimum above 5 without a maximum", give: unboundedSizes(t, 10), want: 20},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, random.Average(tt.give), tt.want, "the average length")
			})
		}
	})

	t.Run("Flag", func(t *testing.T) {
		t.Parallel()

		t.Run("returns true below the minimum without consuming the stream", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(5), random.New(5)
			assert.True(t, random.Flag(&s, sizes(t, 2, 9), 1), "a forced continue")
			assert.Equal(t, s.Next(), twin.Next(), "the next value of the stream")
		})

		t.Run("returns false at the maximum without consuming the stream", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(5), random.New(5)
			assert.False(t, random.Flag(&s, sizes(t, 2, 9), 9), "a forced stop")
			assert.Equal(t, s.Next(), twin.Next(), "the next value of the stream")
		})

		t.Run("continues by a coin of the average extra length in one more", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(5), random.New(5)
			for range 100 {
				assert.Equal(t, random.Flag(&s, unboundedSizes(t, 0), 3), twin.Coin(5, 6), "a coin of 5 in 6")
			}
		})

		t.Run("counts the extra length from the minimum", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(5), random.New(5)
			for range 100 {
				assert.Equal(t, random.Flag(&s, sizes(t, 2, 9), 3), twin.Coin(4, 5), "a coin of 4 in 5")
			}
		})
	})
}

// TestLengthZeroAlloc checks that Average and Flag allocate nothing.
func TestLengthZeroAlloc(t *testing.T) {
	s := random.New(pinnedSeed)
	free := sizes(t, 0, 16)
	assert.MaxAllocs(t, func() { _ = random.Average(free) }, 0, "Average allocates nothing")
	assert.MaxAllocs(t, func() { _ = random.Flag(&s, free, 3) }, 0, "Flag allocates nothing")
}

// BenchmarkLength measures Average and a free Flag under a ceiling of no
// allocation. Every iteration of Flag starts from one state of the stream,
// so the last decision is known.
func BenchmarkLength(b *testing.B) {
	free := sizes(b, 0, 16)

	b.Run("Average", func(b *testing.B) {
		var got int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.Average(free)
		}
		assert.Equal(b, got, 5, "the average length")
	})

	b.Run("Flag", func(b *testing.B) {
		var got bool
		start := random.New(pinnedSeed)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			s := start
			got = random.Flag(&s, free, 3)
		}
		twin := start
		assert.Equal(b, got, twin.Coin(5, 6), "the first decision of the seed")
	})
}

// sizes returns the lengths from minSize to maxSize, failing the test when
// they are invalid.
func sizes(tb testing.TB, minSize, maxSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewSizes(minSize, maxSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}

// unboundedSizes returns the lengths of minSize or more, failing the test
// when they are invalid.
func unboundedSizes(tb testing.TB, minSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewUnboundedSizes(minSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}
