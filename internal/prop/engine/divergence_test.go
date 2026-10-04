// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestDivergence checks the difference that a case tree reports when a
// body requests other choices after the same values, pinned to what the
// definition's executable reference reports.
func TestDivergence(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for the three differences and false past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t,
				[]engine.Difference{engine.RequestDifference, engine.FingerprintDifference, engine.VerdictDifference},
				[]engine.Difference{invalidDifference})
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a request difference for a case that ends where another requested", func(t *testing.T) {
			t.Parallel()
			digit, calls := engine.Integer(0, 9), 0
			got, trace := recorded(func(c *engine.Case) {
				calls++
				if calls == 1 {
					engine.Draw(c, digit, drawn)
				}
			}, settled())
			divergence := &engine.Divergence{
				What:     engine.RequestDifference,
				Recorded: choice.OfInteger(choice.MustIntegerBounds(choice.Int{}, choice.UintOf(9))),
			}
			want := engine.Result{Outcome: engine.Flaky, Cases: 1, Seed: referenceSeed, Divergence: divergence}
			assert.Equal(t, summary(got), want, "the second case ends where the first requested a digit")
			assert.Equal(t, digestOf(trace), "b1a3b096418d4cf6c68e5420a12dfec6089a8f06431456218eacf257b50fbe68",
				"both calls")
		})
	})
}

// TestDivergenceAllocs checks that Valid allocates nothing.
func TestDivergenceAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = engine.VerdictDifference.Valid() }, 0, "Valid allocates nothing")
}

// BenchmarkDivergence measures Valid under a ceiling of no allocation.
func BenchmarkDivergence(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.VerdictDifference.Valid()
		}
		assert.True(b, got, "VerdictDifference is a difference")
	})
}
