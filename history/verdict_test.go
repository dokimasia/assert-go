// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
)

// TestVerdict checks the text of a verdict.
func TestVerdict(t *testing.T) {
	t.Parallel()

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the verdict", func(t *testing.T) {
			t.Parallel()
			got, err := history.Undecided.MarshalText()
			assert.NoError(t, err, "every verdict has a spelling")
			assert.Equal(t, string(got), "undecided", "the spelling of the definition")
		})
	})
}

// TestVerdictAllocs checks that MarshalText allocates its text.
func TestVerdictAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _, _ = history.Undecided.MarshalText() }, 1, "MarshalText allocates its text")
}

// BenchmarkVerdict measures MarshalText.
func BenchmarkVerdict(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got, _ = history.Undecided.MarshalText()
		}
		assert.Equal(b, string(got), "undecided", "the spelling")
	})
}
