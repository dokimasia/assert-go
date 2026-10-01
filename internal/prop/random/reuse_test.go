// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/random"
)

// TestReuse checks the decision whether a reusable draw repeats an earlier
// value of its case.
func TestReuse(t *testing.T) {
	t.Parallel()

	t.Run("Reuse", func(t *testing.T) {
		t.Parallel()

		t.Run("returns false for no earlier value without consuming the stream", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(3), random.New(3)
			_, ok := random.Reuse(&s, 0)
			assert.False(t, ok, "nothing to reuse")
			assert.Equal(t, s.Next(), twin.Next(), "the next value of the stream")
		})

		t.Run("returns the index that a coin of 1 in 4 then a draw below the count choose", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(3), random.New(3)
			for range 200 {
				var want [2]int
				if twin.Coin(1, 4) {
					want = [2]int{int(twin.Below(3)), 1}
				}
				index, ok := random.Reuse(&s, 3)
				got := [2]int{index, 0}
				if ok {
					got[1] = 1
				}
				assert.Equal(t, got, want, "the index and whether the draw reuses it")
			}
		})
	})
}

// TestReuseZeroAlloc checks that Reuse allocates nothing.
func TestReuseZeroAlloc(t *testing.T) {
	s := random.New(pinnedSeed)
	assert.MaxAllocs(t, func() { _, _ = random.Reuse(&s, 3) }, 0, "Reuse allocates nothing")
}

// BenchmarkReuse measures the reuse decision under a ceiling of no
// allocation. Every iteration starts from one state of the stream, so the
// last decision is known.
func BenchmarkReuse(b *testing.B) {
	b.Run("Reuse", func(b *testing.B) {
		var index int
		var ok bool
		start := random.New(pinnedSeed)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			s := start
			index, ok = random.Reuse(&s, 3)
		}
		twin := start
		wantIndex, wantOK := random.Reuse(&twin, 3)
		assert.Equal(b, [2]any{index, ok}, [2]any{wantIndex, wantOK}, "the decision of the first draws")
	})
}
