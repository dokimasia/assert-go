// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// TestRelevanceString pins the spelling of each relevance in the
// definition, and to the engine's for the same value.
func TestRelevanceString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each relevance and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[prop.Relevance]string{
				prop.Untested: "untested", prop.AnyValueFails: "any-value-fails",
				prop.ValueMatters: "value-matters", invalidRelevance: "Relevance(3)",
			})
		})

		t.Run("returns the engine's spelling of the same value", func(t *testing.T) {
			t.Parallel()
			for r := range engine.ValueMatters + 2 {
				assert.Equal(t, prop.Relevance(r).String(), r.String(), "the spelling of value "+r.String())
			}
		})
	})
}

// TestRelevanceStringAllocs checks that String allocates nothing for a
// relevance.
func TestRelevanceStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.ValueMatters.String() }, 0, "String allocates nothing for a relevance")
}

// BenchmarkRelevanceString measures String under a ceiling of no
// allocation.
func BenchmarkRelevanceString(b *testing.B) {
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
