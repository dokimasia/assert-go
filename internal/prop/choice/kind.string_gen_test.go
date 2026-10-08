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

// TestKindString pins the spelling of each kind, which the definition's
// vectors use.
func TestKindString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each kind and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[choice.Kind]string{
				choice.Integer: "integer", choice.Float: "float", choice.Sequence: "sequence", invalidKind: "Kind(3)",
			})
		})
	})
}

// TestKindStringAllocs checks that String allocates nothing for a kind.
func TestKindStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.Float.String() }, 0, "String allocates nothing for a kind")
}

// BenchmarkKindString measures String under a ceiling of no allocation.
func BenchmarkKindString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.Sequence.String()
		}
		assert.Equal(b, got, "sequence", "the kind's spelling")
	})
}
