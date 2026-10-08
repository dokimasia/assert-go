// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/coverage"
)

// TestStageString pins the spelling of each stage.
func TestStageString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of each stage and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[coverage.Stage]string{
				coverage.Interim: "interim", coverage.Final: "final", coverage.Exhausted: "exhausted",
				invalidStage: "Stage(3)",
			})
		})
	})
}

// TestStageStringAllocs checks that String allocates nothing for a stage.
func TestStageStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = coverage.Final.String() }, 0, "String allocates nothing for a stage")
}

// BenchmarkStageString measures String under a ceiling of no allocation.
func BenchmarkStageString(b *testing.B) {
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
