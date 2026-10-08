// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// TestFailure checks that the record's public names have the fields that
// a consumer reads. They are aliases, so the test fails to compile when
// one of them names another type.
func TestFailure(t *testing.T) {
	t.Parallel()

	t.Run("states the assertion, the contract, the detail and the call site", func(t *testing.T) {
		t.Parallel()

		failure := assert.Failure{
			Assertion: "equal",
			Contract:  "the count is right",
			Detail:    map[string]any{"want": 2, "got": 1},
			Where:     assert.Where{File: "store_test.go", Line: 42},
		}

		if got, want := failure.Assertion, "equal"; got != want {
			t.Fatalf("Assertion = %q, want %q", got, want)
		}
		if got, want := failure.Detail["want"], 2; got != want {
			t.Fatalf("Detail[want] = %v, want %v", got, want)
		}
		if got, want := failure.Where.Line, 42; got != want {
			t.Fatalf("Where.Line = %d, want %d", got, want)
		}
	})

	t.Run("a Recorder is a Reporter", func(t *testing.T) {
		t.Parallel()

		// An assertion passes the record only to a seat that satisfies
		// Reporter, and no other test checks that a Recorder does.
		var _ assert.Reporter = assert.NewRecorder()
	})
}
