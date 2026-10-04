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

// TestKind checks which values are kinds of events, and their text.
func TestKind(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for each of the four kinds and false for the values around them", func(t *testing.T) {
			t.Parallel()
			enumtest.Members(t, []history.Kind{history.Invoke, history.OK, history.Fail, history.Unknown},
				[]history.Kind{0, invalidKind})
		})
	})

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the kind", func(t *testing.T) {
			t.Parallel()
			got, err := history.Unknown.MarshalText()
			assert.NoError(t, err, "every kind has a spelling")
			assert.Equal(t, string(got), "unknown", "the spelling of the definition")
		})
	})
}

// TestKindAllocs checks that Valid allocates nothing, and that MarshalText
// allocates its text.
func TestKindAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = history.Unknown.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = history.Unknown.MarshalText() }, 1, "MarshalText allocates its text")
}

// BenchmarkKind measures Valid under a ceiling of no allocation, and
// MarshalText.
func BenchmarkKind(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got, _ = history.Unknown.MarshalText()
		}
		assert.Equal(b, string(got), "unknown", "the spelling")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = history.Unknown.Valid()
		}
		assert.True(b, got, "Unknown is a kind")
	})
}
