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

// TestRelationString pins the spelling of each relation in the definition.
func TestRelationString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each relation and stringer's around them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[history.Relation]string{
				history.WW: "ww", history.WR: "wr", history.RW: "rw",
				0: "Relation(0)", invalidRelation: "Relation(4)",
			})
		})
	})
}

// TestRelationStringAllocs checks that String allocates nothing for a
// relation.
func TestRelationStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = history.RW.String() }, 0, "String allocates nothing for a relation")
}

// BenchmarkRelationString measures String under a ceiling of no
// allocation.
func BenchmarkRelationString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = history.RW.String()
		}
		assert.Equal(b, got, "rw", "the relation's spelling")
	})
}
