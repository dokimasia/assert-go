// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/enumtest"
)

// TestLimitString pins the spelling of each limit in the definition.
func TestLimitString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's spelling of each limit and stringer's around them", func(t *testing.T) {
			t.Parallel()
			enumtest.Spellings(t, map[history.Limit]string{
				history.LimitSteps: "steps", history.LimitMemo: "memo", history.LimitTime: "time",
				0: "Limit(0)", invalidLimit: "Limit(4)",
			})
		})
	})
}

// TestLimitStringAllocs checks that String allocates nothing for a limit.
func TestLimitStringAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = history.LimitTime.String() }, 0, "String allocates nothing for a limit")
}

// BenchmarkLimitString measures String under a ceiling of no allocation.
func BenchmarkLimitString(b *testing.B) {
	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = history.LimitTime.String()
		}
		assert.Equal(b, got, "time", "the limit's spelling")
	})
}
