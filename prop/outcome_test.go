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

// invalidOutcome is the first value past the six outcomes.
const invalidOutcome prop.Outcome = 6

// TestOutcome checks which values are outcomes, and pins each outcome's
// spelling to the definition's and to the engine's for the same value.
func TestOutcome(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Outcome
			want bool
		}{
			{name: "reports true for Passed", give: prop.Passed, want: true},
			{name: "reports true for Vacuous", give: prop.Vacuous, want: true},
			{name: "reports false past Vacuous", give: invalidOutcome, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is an outcome")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Outcome
			want string
		}{
			{name: "returns passed for Passed", give: prop.Passed, want: "passed"},
			{name: "returns counterexample for Counterexample", give: prop.Counterexample, want: "counterexample"},
			{name: "returns flaky for Flaky", give: prop.Flaky, want: "flaky"},
			{name: "returns rejected for Rejected", give: prop.Rejected, want: "rejected"},
			{name: "returns coverage-unmet for CoverageUnmet", give: prop.CoverageUnmet, want: "coverage-unmet"},
			{name: "returns vacuous for Vacuous", give: prop.Vacuous, want: "vacuous"},
			{name: "returns Outcome(6) for a value that is no outcome", give: invalidOutcome, want: "Outcome(6)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the outcome's spelling")
			})
		}

		t.Run("returns the engine's spelling of the same value", func(t *testing.T) {
			t.Parallel()
			for o := range engine.Vacuous + 1 {
				assert.Equal(t, prop.Outcome(o).String(), o.String(), "the spelling of value "+o.String())
			}
		})
	})
}

// TestOutcomeZeroAlloc checks that no method of Outcome allocates.
func TestOutcomeZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.Vacuous.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = prop.Vacuous.String() }, 0, "String allocates nothing")
}

// BenchmarkOutcome measures each method of Outcome under a ceiling of no
// allocation.
func BenchmarkOutcome(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Vacuous.Valid()
		}
		assert.True(b, got, "Vacuous is an outcome")
	})

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
