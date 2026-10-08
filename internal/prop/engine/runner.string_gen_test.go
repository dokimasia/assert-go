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

// TestOutcomeString pins the spelling of each outcome in the definition.
func TestOutcomeString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each outcome and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[engine.Outcome]string{
				engine.Passed: "passed", engine.Counterexample: "counterexample", engine.Flaky: "flaky",
				engine.Rejected: "rejected", engine.CoverageUnmet: "coverage-unmet", engine.Vacuous: "vacuous",
				invalidOutcome: "Outcome(6)",
			})
		})
	})
}

// TestOutcomeStringAllocs checks that String allocates nothing for an
// outcome.
func TestOutcomeStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = engine.Vacuous.String() }, 0, "String allocates nothing for an outcome")
}

// BenchmarkOutcomeString measures String under a ceiling of no allocation.
func BenchmarkOutcomeString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.Vacuous.String()
		}
		assert.Equal(b, got, "vacuous", "the outcome's spelling")
	})
}
