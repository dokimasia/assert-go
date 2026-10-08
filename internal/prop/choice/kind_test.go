// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestKind checks which values are kinds.
func TestKind(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for the three kinds and false past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t, []choice.Kind{choice.Integer, choice.Float, choice.Sequence},
				[]choice.Kind{invalidKind})
		})
	})
}

// TestKindAllocs checks that Valid allocates nothing.
func TestKindAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.Float.Valid() }, 0, "Valid allocates nothing")
}

// BenchmarkKind measures Valid under a ceiling of no allocation.
func BenchmarkKind(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.Sequence.Valid()
		}
		assert.True(b, got, "Sequence is a kind")
	})
}
