// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/random"
)

// keptActions is the number of actions from the one that a pinned keep
// decides to the last.
const keptActions = 3

// TestKeep checks the swarm's decisions whether to keep an action.
func TestKeep(t *testing.T) {
	t.Parallel()

	t.Run("Keep", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			kept bool
			want []bool
		}{
			{
				name: "returns a coin of the odds once an earlier action is kept",
				kept: true,
				want: []bool{false, true, true, false, false, true, true, true, false, true, true, false},
			},
			{
				name: "returns its own coin of the first round in which a coin of the remaining actions came up",
				want: []bool{false, false, true, true, true, true, true, false, false, false, true, false},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := random.New(pinnedSeed)
				got := make([]bool, len(tt.want))
				for i := range got {
					got[i] = random.Keep(&s, 1, 2, tt.kept, keptActions)
				}
				assert.Equal(t, got, tt.want, "the first decisions of the seed")
			})
		}

		t.Run("keeps the last action without consuming the stream while none is kept", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(pinnedSeed), random.New(pinnedSeed)
			assert.True(t, random.Keep(&s, 1, 2, false, 1), "the last action")
			assert.Equal(t, s.Next(), twin.Next(), "the next value of the stream")
		})

		t.Run("tosses the coins of every remaining action in each round", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(pinnedSeed), random.New(pinnedSeed)
			random.Keep(&s, 1, 2, false, keptActions)
			for some := false; !some; {
				for range keptActions {
					some = twin.Coin(1, 2) || some
				}
			}
			assert.Equal(t, s.Next(), twin.Next(), "the stream after the round in which a coin came up")
		})
	})
}

// TestKeepAllocs checks that Keep allocates nothing.
func TestKeepAllocs(t *testing.T) {
	s := random.New(pinnedSeed)
	assert.MaxAllocs(t, func() { _ = random.Keep(&s, 1, 2, false, keptActions) }, 0, "Keep allocates nothing")
}

// BenchmarkKeep measures a decision while no earlier action is kept, under a
// ceiling of no allocation. Every iteration starts from one state of the
// stream, so the decision is known.
func BenchmarkKeep(b *testing.B) {
	b.Run("Keep", func(b *testing.B) {
		var got bool
		start := random.New(pinnedSeed)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			s := start
			got = random.Keep(&s, 1, 2, false, keptActions)
		}
		assert.False(b, got, "the first decision of the seed")
	})
}
