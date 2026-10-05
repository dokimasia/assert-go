// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/enumtest"
)

// TestAnomalyString pins the spelling of each kind of anomaly in the
// definition.
func TestAnomalyString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each kind and stringer's around them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[history.Anomaly]string{
				history.GarbageRead:           "garbage-read",
				history.DuplicateAppend:       "duplicate-append",
				history.InternalInconsistency: "internal-inconsistency",
				history.IncompatibleOrder:     "incompatible-order",
				history.AbortedRead:           "aborted-read",
				history.IntermediateRead:      "intermediate-read",
				history.G0:                    "G0",
				history.G1c:                   "G1c",
				history.GSingle:               "G-single",
				history.GNonadjacent:          "G-nonadjacent",
				history.G2:                    "G2",
				0:                             "Anomaly(0)",
				invalidAnomaly:                "Anomaly(12)",
			})
		})
	})
}

// TestAnomalyStringAllocs checks that String allocates nothing for a kind.
func TestAnomalyStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = history.G2.String() }, 0, "String allocates nothing for a kind")
}

// BenchmarkAnomalyString measures String under a ceiling of no allocation.
func BenchmarkAnomalyString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = history.G2.String()
		}
		assert.Equal(b, got, "G2", "the kind's spelling")
	})
}
