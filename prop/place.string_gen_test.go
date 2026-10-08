// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/prop"
)

// TestPartString pins the spelling of each part in the definition.
func TestPartString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each part and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[prop.Part]string{
				prop.SwarmPart: "swarm", prop.SetupPart: "setup", prop.SequentialPart: "sequential",
				prop.ConcurrentPart: "concurrent", prop.DrainPart: "drain", prop.SettlePart: "settle",
				invalidPart: "Part(6)",
			})
		})
	})
}

// TestPartStringAllocs checks that String allocates nothing for a part.
func TestPartStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.SettlePart.String() }, 0, "String allocates nothing for a part")
}

// BenchmarkPartString measures String under a ceiling of no allocation.
func BenchmarkPartString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.SettlePart.String()
		}
		assert.Equal(b, got, "settle", "the part's spelling")
	})
}
