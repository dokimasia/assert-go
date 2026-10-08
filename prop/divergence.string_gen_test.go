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

// TestDifferenceString pins the spelling of each difference in the
// definition, and to the engine's for the same value.
func TestDifferenceString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each difference and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[prop.Difference]string{
				prop.RequestDifference: "request", prop.FingerprintDifference: "fingerprint",
				prop.VerdictDifference: "verdict", invalidDifference: "Difference(3)",
			})
		})

		t.Run("returns the engine's spelling of the same value", func(t *testing.T) {
			t.Parallel()
			for d := range engine.VerdictDifference + 2 {
				assert.Equal(t, prop.Difference(d).String(), d.String(), "the spelling of value "+d.String())
			}
		})
	})
}

// TestDifferenceStringAllocs checks that String allocates nothing for a
// difference.
func TestDifferenceStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.VerdictDifference.String() }, 0,
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
			got = prop.VerdictDifference.String()
		}
		assert.Equal(b, got, "verdict", "the difference's spelling")
	})
}
