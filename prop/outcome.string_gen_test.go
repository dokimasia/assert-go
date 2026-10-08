// Copyright Dokimasia B.V. 2026
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

// TestOutcomeString pins the spelling of each outcome in the definition,
// and to the engine's for the same value.
func TestOutcomeString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each outcome and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[prop.Outcome]string{
				prop.Passed: "passed", prop.Counterexample: "counterexample", prop.Flaky: "flaky",
				prop.Rejected: "rejected", prop.CoverageUnmet: "coverage-unmet", prop.Vacuous: "vacuous",
				invalidOutcome: "Outcome(6)",
			})
		})

		t.Run("returns the engine's spelling of the same value", func(t *testing.T) {
			t.Parallel()
			for o := range engine.Vacuous + 2 {
				assert.Equal(t, prop.Outcome(o).String(), o.String(), "the spelling of value "+o.String())
			}
		})
	})
}

// TestOutcomeStringAllocs checks that String allocates nothing for an
// outcome.
func TestOutcomeStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.Vacuous.String() }, 0, "String allocates nothing for an outcome")
}

// BenchmarkOutcomeString measures String under a ceiling of no allocation.
func BenchmarkOutcomeString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Vacuous.String()
		}
		assert.Equal(b, got, "vacuous", "the outcome's spelling")
	})
}
