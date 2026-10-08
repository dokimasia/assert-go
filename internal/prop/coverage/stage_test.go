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

// TestStage checks which values are stages.
func TestStage(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for the three stages and false past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t, []coverage.Stage{coverage.Interim, coverage.Final, coverage.Exhausted},
				[]coverage.Stage{invalidStage})
		})
	})
}

// TestStageAllocs checks that Valid allocates nothing.
func TestStageAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = coverage.Final.Valid() }, 0, "Valid allocates nothing")
}

// BenchmarkStage measures Valid under a ceiling of no allocation.
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
}
