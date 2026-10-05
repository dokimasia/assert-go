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

// TestRelation checks which values are relations, and their text.
func TestRelation(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for each of the three relations and false for the values around them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t, []history.Relation{history.WW, history.WR, history.RW},
				[]history.Relation{0, invalidRelation})
		})
	})

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the relation", func(t *testing.T) {
			t.Parallel()
			got, err := history.RW.MarshalText()
			assert.NoError(t, err, "every relation has a spelling")
			assert.Equal(t, string(got), "rw", "the spelling of the definition")
		})
	})
}

// TestRelationAllocs checks that Valid allocates nothing, and that
// MarshalText allocates its text.
func TestRelationAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = history.RW.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = history.RW.MarshalText() }, 1, "MarshalText allocates its text")
}

// BenchmarkRelation measures Valid under a ceiling of no allocation, and
// MarshalText.
func BenchmarkRelation(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got, _ = history.RW.MarshalText()
		}
		assert.Equal(b, string(got), "rw", "the spelling")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = history.RW.Valid()
		}
		assert.True(b, got, "RW is a relation")
	})
}
