// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/coverage"
)

// invalidStage is the first value past the three stages.
const invalidStage coverage.Stage = 3

// TestStage checks which values are stages, and pins each stage's
// spelling.
func TestStage(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give coverage.Stage
			want bool
		}{
			{name: "reports true for Interim", give: coverage.Interim, want: true},
			{name: "reports true for Exhausted", give: coverage.Exhausted, want: true},
			{name: "reports false past Exhausted", give: invalidStage, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a stage")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give coverage.Stage
			want string
		}{
			{name: "returns interim for Interim", give: coverage.Interim, want: "interim"},
			{name: "returns final for Final", give: coverage.Final, want: "final"},
			{name: "returns exhausted for Exhausted", give: coverage.Exhausted, want: "exhausted"},
			{name: "returns Stage(3) for a value that is no stage", give: invalidStage, want: "Stage(3)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the stage's spelling")
			})
		}
	})
}

// TestStageZeroAlloc checks that no method of Stage allocates.
func TestStageZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = coverage.Final.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = coverage.Final.String() }, 0, "String allocates nothing")
}

// BenchmarkStage measures each method of Stage under a ceiling of no
// allocation.
func BenchmarkStage(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = coverage.Final.Valid()
		}
		assert.True(b, got, "Final is a stage")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = coverage.Final.String()
		}
		assert.Equal(b, got, "final", "the stage's spelling")
	})
}
