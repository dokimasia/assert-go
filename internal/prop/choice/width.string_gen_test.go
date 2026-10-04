// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestWidthString pins the spelling of each width.
func TestWidthString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the bits of each width and stringer's spelling of any other value", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[choice.Width]string{
				choice.Width32: "32", choice.Width64: "64", 16: "Width(16)",
			})
		})
	})
}

// TestWidthStringAllocs checks that String allocates nothing for a
// width.
func TestWidthStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = choice.Width32.String() }, 0, "String allocates nothing for a width")
}

// BenchmarkWidthString measures String under a ceiling of no allocation.
func BenchmarkWidthString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.Width64.String()
		}
		assert.Equal(b, got, "64", "the width's spelling")
	})
}
