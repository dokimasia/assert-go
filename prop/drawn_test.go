// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// invalidRelevance is the first value past the three relevances.
const invalidRelevance prop.Relevance = 3

// TestDrawn checks the draws of a counterexample, pinned to the
// definition's behaviour vectors, and pins each relevance's spelling.
func TestDrawn(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Relevance
			want bool
		}{
			{name: "reports true for Untested", give: prop.Untested, want: true},
			{name: "reports true for ValueMatters", give: prop.ValueMatters, want: true},
			{name: "reports false past ValueMatters", give: invalidRelevance, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a relevance")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Relevance
			want string
		}{
			{name: "returns untested for Untested", give: prop.Untested, want: "untested"},
			{name: "returns any-value-fails for AnyValueFails", give: prop.AnyValueFails, want: "any-value-fails"},
			{name: "returns value-matters for ValueMatters", give: prop.ValueMatters, want: "value-matters"},
			{
				name: "returns Relevance(3) for a value that is no relevance",
				give: invalidRelevance,
				want: "Relevance(3)",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the relevance's spelling")
			})
		}

		t.Run("returns the engine's spelling of the same value", func(t *testing.T) {
			t.Parallel()
			for r := range engine.ValueMatters + 1 {
				assert.Equal(t, prop.Relevance(r).String(), r.String(), "the spelling of value "+r.String())
			}
		})
	})

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the nearest passing value of an integer whose value matters", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7))
			want := []prop.Drawn{{Label: drawn, Value: 1001, Relevance: prop.ValueMatters, NearestPassing: 1000}}
			assert.Equal(t, got[counterexampleField], any(want), "the minimal value and the one below it")
		})

		t.Run("reports a draw where any value fails", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(100, 0, every), prop.Seed(7))
			want := []prop.Drawn{{Label: drawn, Value: 0, Relevance: prop.AnyValueFails}}
			assert.Equal(t, got[counterexampleField], any(want), "the target, where every filling fails")
		})

		t.Run("reports a draw whose rejected fillings are skipped as one where any value fails", func(t *testing.T) {
			t.Parallel()
			unique := prop.List(prop.Integer(0, 2), prop.Unique(), prop.MinSize(3), prop.MaxSize(3))
			body := func(c *prop.Case) {
				c.Draw(unique, drawn)
				fail(c, always)
			}
			got := detailOf(body, prop.Seed(0))
			want := []prop.Drawn{{Label: drawn, Value: []int{0, 1, 2}, Relevance: prop.AnyValueFails}}
			assert.Equal(t, got[counterexampleField], any(want), "the one list of three distinct digits")
		})

		t.Run("leaves a draw untested when no filling decodes", func(t *testing.T) {
			t.Parallel()
			seven := prop.Integer(0, 9).Filter(func(v int) bool { return v == 7 })
			body := func(c *prop.Case) {
				c.Draw(seven, drawn)
				fail(c, "seven")
			}
			got := detailOf(body, prop.Seed(2))
			want := []prop.Drawn{{Label: drawn, Value: 7}}
			assert.Equal(t, got[counterexampleField], any(want), "the kept value, untested")
		})

		t.Run("leaves the draws untested when shrinking is off", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Shrink(0))
			want := []prop.Drawn{{Label: drawn, Value: 8522}}
			assert.Equal(t, got[counterexampleField], any(want), "the first failing value, as found")
		})

		t.Run("leaves the draws untested when explaining is off", func(t *testing.T) {
			t.Parallel()
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Explain(false))
			want := []prop.Drawn{{Label: drawn, Value: 1001}}
			assert.Equal(t, got[counterexampleField], any(want), "the minimal value, unexplained")
		})
	})
}

// TestDrawnZeroAlloc checks that no method of Relevance allocates.
func TestDrawnZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.ValueMatters.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = prop.ValueMatters.String() }, 0, "String allocates nothing")
}

// BenchmarkDrawn measures each method of Relevance under a ceiling of no
// allocation.
func BenchmarkDrawn(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.ValueMatters.Valid()
		}
		assert.True(b, got, "ValueMatters is a relevance")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.ValueMatters.String()
		}
		assert.Equal(b, got, "value-matters", "the relevance's spelling")
	})
}
