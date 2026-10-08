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

// TestStatusString pins the spelling of each status of a case.
func TestStatusString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of each status and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[engine.Status]string{
				engine.CasePassed: "passed", engine.CaseFailed: "failed", engine.CaseRejected: "rejected",
				engine.CaseRepeated: "repeated", engine.CaseDiverged: "diverged", engine.CaseRefused: "refused",
				invalidStatus: "Status(6)",
			})
		})
	})
}

// TestStatusStringAllocs checks that String allocates nothing for a
// status.
func TestStatusStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = engine.CaseDiverged.String() }, 0, "String allocates nothing for a status")
}

// BenchmarkStatusString measures String under a ceiling of no allocation.
func BenchmarkStatusString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.CaseDiverged.String()
		}
		assert.Equal(b, got, "diverged", "the status's spelling")
	})
}
