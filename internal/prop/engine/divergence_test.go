// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// invalidDifference is the first value past the three differences.
const invalidDifference engine.Difference = 3

// TestDivergence checks the difference that a case tree reports when a
// body requests other choices after the same values, pinned to what the
// definition's executable reference reports, and pins each difference's
// spelling.
func TestDivergence(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give engine.Difference
			want bool
		}{
			{name: "reports true for RequestDifference", give: engine.RequestDifference, want: true},
			{name: "reports true for VerdictDifference", give: engine.VerdictDifference, want: true},
			{name: "reports false past VerdictDifference", give: invalidDifference, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a difference")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give engine.Difference
			want string
		}{
			{name: "returns request for RequestDifference", give: engine.RequestDifference, want: "request"},
			{
				name: "returns fingerprint for FingerprintDifference",
				give: engine.FingerprintDifference,
				want: "fingerprint",
			},
			{name: "returns verdict for VerdictDifference", give: engine.VerdictDifference, want: "verdict"},
			{
				name: "returns Difference(3) for a value that is no difference",
				give: invalidDifference,
				want: "Difference(3)",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the difference's spelling")
			})
		}
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

// TestDivergenceZeroAlloc checks that no method of Difference allocates.
func TestDivergenceZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = engine.VerdictDifference.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = engine.VerdictDifference.String() }, 0, "String allocates nothing")
}

// BenchmarkDivergence measures each method of Difference under a ceiling
// of no allocation.
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

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.VerdictDifference.String()
		}
		assert.Equal(b, got, "verdict", "the difference's spelling")
	})
}
