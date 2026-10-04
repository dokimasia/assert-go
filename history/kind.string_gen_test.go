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

// TestKindString pins the spelling of each kind of event in the definition.
func TestKindString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each kind and stringer's around them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[history.Kind]string{
				history.Invoke: "invoke", history.OK: "ok", history.Fail: "fail", history.Unknown: "unknown",
				0: "Kind(0)", invalidKind: "Kind(5)",
			})
		})
	})
}

// TestKindStringAllocs checks that String allocates nothing for a kind.
func TestKindStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = history.Unknown.String() }, 0, "String allocates nothing for a kind")
}

// BenchmarkKindString measures String under a ceiling of no allocation.
func BenchmarkKindString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = history.Unknown.String()
		}
		assert.Equal(b, got, "unknown", "the kind's spelling")
	})
}
