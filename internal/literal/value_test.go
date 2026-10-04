// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package literal_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/literal"
)

// TestValue checks how the values of the record, map and enum shapes
// compare as Go values.
func TestValue(t *testing.T) {
	t.Parallel()

	t.Run("Record", func(t *testing.T) {
		t.Parallel()

		t.Run("compares its fields in declaration order", func(t *testing.T) {
			t.Parallel()
			record := literal.Record{Fields: []literal.Field{{Name: "id", Value: 1}, {Name: "note"}}}
			assert.Equal(t, record, literal.Record{Fields: []literal.Field{{Name: "id", Value: 1}, {Name: "note"}}},
				"the same fields in the same order")
			assert.NotEqual(t, record, literal.Record{Fields: []literal.Field{{Name: "note"}, {Name: "id", Value: 1}}},
				"the same fields in another order")
		})
	})

	t.Run("Pairs", func(t *testing.T) {
		t.Parallel()

		t.Run("compares its entries in the order of generation", func(t *testing.T) {
			t.Parallel()
			pairs := literal.Pairs{Entries: []literal.Entry{{Key: 1, Value: "a"}, {Key: 2, Value: "b"}}}
			reordered := literal.Pairs{Entries: []literal.Entry{{Key: 2, Value: "b"}, {Key: 1, Value: "a"}}}
			assert.NotEqual(t, pairs, reordered, "the same entries in another order")
		})
	})

	t.Run("Variant", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a variant without a payload apart from one whose payload is absent", func(t *testing.T) {
			t.Parallel()
			assert.NotEqual(t, literal.Variant{Name: "refunded"}, literal.Variant{Name: "refunded", HasPayload: true},
				"no payload and an absent payload")
		})
	})
}
