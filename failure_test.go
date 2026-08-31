// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// TestFailure holds the record's public names to the shapes a consumer
// reads them at. They are aliases, so this is what stops one being
// re-pointed at something else without anyone noticing.
func TestFailure(t *testing.T) {
	t.Parallel()

	t.Run("carries the assertion, the contract and the detail", func(t *testing.T) {
		t.Parallel()

		held := assert.Failure{
			Assertion: "equal",
			Contract:  "the count is right",
			Detail:    map[string]any{"want": 2, "got": 1},
			Where:     assert.Where{File: "store_test.go", Line: 42},
		}

		if got, want := held.Assertion, "equal"; got != want {
			t.Fatalf("Assertion = %q, want %q", got, want)
		}
		if got, want := held.Detail["want"], 2; got != want {
			t.Fatalf("Detail[want] = %v, want %v", got, want)
		}
		if got, want := held.Where.Line, 42; got != want {
			t.Fatalf("Where.Line = %d, want %d", got, want)
		}
	})

	t.Run("a Recorder is a Reporter", func(t *testing.T) {
		t.Parallel()

		// The optional upgrade only works if the seat satisfies it, and
		// nothing else states that it does.
		var _ assert.Reporter = assert.NewRecorder()
	})
}
