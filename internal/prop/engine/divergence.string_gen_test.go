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

// TestDifferenceString pins the spelling of each difference in the
// definition.
func TestDifferenceString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each difference and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[engine.Difference]string{
				engine.RequestDifference: "request", engine.FingerprintDifference: "fingerprint",
				engine.VerdictDifference: "verdict", invalidDifference: "Difference(3)",
			})
		})
	})
}

// TestDifferenceStringAllocs checks that String allocates nothing for a
// difference.
func TestDifferenceStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = engine.VerdictDifference.String() }, 0,
		"String allocates nothing for a difference")
}

// BenchmarkDifferenceString measures String under a ceiling of no
// allocation.
func BenchmarkDifferenceString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.VerdictDifference.String()
		}
		assert.Equal(b, got, "verdict", "the difference's spelling")
	})
}
