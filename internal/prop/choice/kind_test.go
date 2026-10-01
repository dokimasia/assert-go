// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// invalidKind is the first value past the three kinds.
const invalidKind = choice.Kind(3)

// TestKind checks which values are kinds, and pins each kind's spelling,
// which the definition's vectors use.
func TestKind(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for each of the three kinds", func(t *testing.T) {
			t.Parallel()
			for _, k := range []choice.Kind{choice.Integer, choice.Float, choice.Sequence} {
				assert.True(t, k.Valid(), k.String()+" is a kind")
			}
		})

		t.Run("reports false past the three kinds", func(t *testing.T) {
			t.Parallel()
			assert.False(t, invalidKind.Valid(), "kind 3 is no kind")
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Kind
			want string
		}{
			{name: "returns integer for Integer", give: choice.Integer, want: "integer"},
			{name: "returns float for Float", give: choice.Float, want: "float"},
			{name: "returns sequence for Sequence", give: choice.Sequence, want: "sequence"},
			{name: "returns Kind(3) for a value past the kinds", give: invalidKind, want: "Kind(3)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the kind's spelling")
			})
		}
	})
}

// TestKindZeroAlloc checks that no method of Kind allocates.
func TestKindZeroAlloc(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.Float.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.Float.String() }, 0, "String allocates nothing")
}

// BenchmarkKind measures each method of Kind under a ceiling of no
// allocation.
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
