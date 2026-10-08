// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/store"
)

// TestVerdict checks which values are verdicts.
func TestVerdict(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for the four verdicts and false past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t, []store.Verdict{store.Replay, store.Other, store.Skip, store.Damaged},
				[]store.Verdict{invalidVerdict})
		})
	})
}

// TestVerdictAllocs checks that Valid allocates nothing.
func TestVerdictAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = store.Damaged.Valid() }, 0, "Valid allocates nothing")
}

// BenchmarkVerdict measures Valid under a ceiling of no allocation.
func BenchmarkVerdict(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = store.Damaged.Valid()
		}
		assert.True(b, got, "Damaged is a verdict")
	})
}
