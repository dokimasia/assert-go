// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/enumtest"
)

// TestAnomaly checks which values are kinds of anomaly, and their text.
func TestAnomaly(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for each of the eleven kinds and false for the values around them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t, []history.Anomaly{
				history.GarbageRead, history.DuplicateAppend, history.InternalInconsistency,
				history.IncompatibleOrder, history.AbortedRead, history.IntermediateRead, history.G0,
				history.G1c, history.GSingle, history.GNonadjacent, history.G2,
			}, []history.Anomaly{0, invalidAnomaly})
		})
	})

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the kind", func(t *testing.T) {
			t.Parallel()
			got, err := history.GNonadjacent.MarshalText()
			assert.NoError(t, err, "every kind has a spelling")
			assert.Equal(t, string(got), "G-nonadjacent", "the spelling of the definition")
		})
	})
}

// TestAnomalyAllocs checks that Valid allocates nothing, and that
// MarshalText allocates its text.
func TestAnomalyAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = history.G2.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = history.G2.MarshalText() }, 2, "MarshalText allocates its text")
}

// BenchmarkAnomaly measures Valid under a ceiling of no allocation, and
// MarshalText.
func BenchmarkAnomaly(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(2)
		defer c.End()
		for c.Loop() {
			got, _ = history.G2.MarshalText()
		}
		assert.Equal(b, string(got), "G2", "the spelling")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = history.G2.Valid()
		}
		assert.True(b, got, "G2 is a kind")
	})
}
