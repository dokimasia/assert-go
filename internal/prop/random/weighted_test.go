// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/random"
)

// pinnedWeights are the weights of the pinned draws by weight.
var pinnedWeights = []uint64{1, 2, 3}

// TestWeighted checks the draw of an index by weight.
func TestWeighted(t *testing.T) {
	t.Parallel()

	t.Run("Weighted", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the pinned indices of the weights 1, 2 and 3", func(t *testing.T) {
			t.Parallel()
			s := random.New(pinnedSeed)
			got := make([]int, 12)
			for i := range got {
				got[i] = random.Weighted(&s, pinnedWeights)
			}
			assert.Equal(t, got, []int{2, 2, 1, 2, 1, 1, 1, 2, 0, 1, 1, 2}, "the first draws of the seed")
		})

		t.Run("returns the first index whose running sum exceeds one draw below the sum", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(pinnedSeed), random.New(pinnedSeed)
			for range draws {
				point, want := twin.Below(6), 2
				if point < 1 {
					want = 0
				} else if point < 3 {
					want = 1
				}
				assert.Equal(t, random.Weighted(&s, pinnedWeights), want, "the index of the draw's weight")
			}
		})

		t.Run("returns 0 for one weight without consuming the stream", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(pinnedSeed), random.New(pinnedSeed)
			assert.Equal(t, random.Weighted(&s, []uint64{5}), 0, "the one index")
			assert.Equal(t, s.Next(), twin.Next(), "the next value of the stream")
		})
	})
}

// TestWeightedAllocs checks that Weighted allocates nothing.
func TestWeightedAllocs(t *testing.T) {
	s := random.New(pinnedSeed)
	assert.MaxAllocs(t, func() { _ = random.Weighted(&s, pinnedWeights) }, 0, "Weighted allocates nothing")
}

// BenchmarkWeighted measures a draw among three weights under a ceiling of
// no allocation. Every iteration starts from one state of the stream, so
// the index is known.
func BenchmarkWeighted(b *testing.B) {
	b.Run("Weighted", func(b *testing.B) {
		var got int
		start := random.New(pinnedSeed)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			s := start
			got = random.Weighted(&s, pinnedWeights)
		}
		assert.Equal(b, got, 2, "the first draw of the seed")
	})
}
