// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestRelevanceString pins the spelling of each relevance in the
// definition.
func TestRelevanceString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each relevance and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[engine.Relevance]string{
				engine.Untested: "untested", engine.AnyValueFails: "any-value-fails",
				engine.ValueMatters: "value-matters", invalidRelevance: "Relevance(3)",
			})
		})
	})
}

// TestRelevanceStringAllocs checks that String allocates nothing for a
// relevance.
func TestRelevanceStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = engine.ValueMatters.String() }, 0, "String allocates nothing for a relevance")
}

// BenchmarkRelevanceString measures String under a ceiling of no
// allocation.
func BenchmarkRelevanceString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.ValueMatters.String()
		}
		assert.Equal(b, got, "value-matters", "the relevance's spelling")
	})
}
