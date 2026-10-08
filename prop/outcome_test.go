// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// TestOutcome checks which values are outcomes.
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

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the outcome", func(t *testing.T) {
			t.Parallel()
			got, err := prop.CoverageUnmet.MarshalText()
			assert.NoError(t, err, "every outcome has a spelling")
			assert.Equal(t, string(got), "coverage-unmet", "the spelling of the definition")
		})
	})
}

// TestOutcomeAllocs checks that Valid allocates nothing, and that
// MarshalText allocates its text.
func TestOutcomeAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.Vacuous.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = prop.Vacuous.MarshalText() }, 2, "MarshalText allocates its text")
}

// BenchmarkOutcome measures Valid under a ceiling of no allocation, and
// MarshalText.
func BenchmarkOutcome(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(2)
		defer c.End()
		for c.Loop() {
			got, _ = prop.Vacuous.MarshalText()
		}
		assert.Equal(b, string(got), "vacuous", "the spelling")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Vacuous.Valid()
		}
		assert.True(b, got, "Vacuous is an outcome")
	})
}
