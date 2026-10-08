// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestBehaviour(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "reports a match of an error with context.Canceled as HonoursCancellation",
				give: "./honours-cancellation",
			},
			{
				name: "reports a match of an error with context.DeadlineExceeded as HonoursDeadline",
				give: "./honours-deadline",
			},
			{name: "reports a select whose time.After fails the test as CompletesWithin", give: "./completes-within"},
			{name: "reports a nil context in a test file as NilContextSafe", give: "./nil-context-safe"},
		}
		analyzeEach(t, tests)
	})
}
