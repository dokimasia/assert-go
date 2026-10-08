// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
)

// TestLimit checks the text of a limit.
func TestLimit(t *testing.T) {
	t.Parallel()

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the limit", func(t *testing.T) {
			t.Parallel()
			got, err := history.LimitMemo.MarshalText()
			assert.NoError(t, err, "every limit has a spelling")
			assert.Equal(t, string(got), "memo", "the spelling of the definition")
		})
	})
}

// TestLimitAllocs checks that MarshalText allocates its text.
func TestLimitAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _, _ = history.LimitMemo.MarshalText() }, 2, "MarshalText allocates its text")
}

// BenchmarkLimit measures MarshalText.
func BenchmarkLimit(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(2)
		defer c.End()
		for c.Loop() {
			got, _ = history.LimitMemo.MarshalText()
		}
		assert.Equal(b, string(got), "memo", "the spelling")
	})
}
