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

// TestVerdictString pins the spelling of each verdict in the definition.
func TestVerdictString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each verdict and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[coverage.Verdict]string{
				coverage.Met: "met", coverage.Refuted: "refuted", coverage.Undecided: "undecided",
				coverage.Unmet: "unmet", invalidVerdict: "Verdict(4)",
			})
		})
	})
}

// TestVerdictStringAllocs checks that String allocates nothing for a
// verdict.
func TestVerdictStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = coverage.Unmet.String() }, 0, "String allocates nothing for a verdict")
}

// BenchmarkVerdictString measures String under a ceiling of no allocation.
func BenchmarkVerdictString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = coverage.Unmet.String()
		}
		assert.Equal(b, got, "unmet", "the verdict's spelling")
	})
}
