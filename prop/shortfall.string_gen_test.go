// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/prop"
)

// TestVerdictString pins the spelling of each verdict of a coverage
// requirement in the definition.
func TestVerdictString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each verdict and stringer's past them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[prop.Verdict]string{
				prop.Refuted: "refuted", prop.Unmet: "unmet", invalidVerdict: "Verdict(2)",
			})
		})
	})
}

// TestVerdictStringAllocs checks that String allocates nothing for a
// verdict.
func TestVerdictStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.Unmet.String() }, 0, "String allocates nothing for a verdict")
}

// BenchmarkVerdictString measures String under a ceiling of no allocation.
func BenchmarkVerdictString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.Unmet.String()
		}
		assert.Equal(b, got, "unmet", "the verdict's spelling")
	})
}
