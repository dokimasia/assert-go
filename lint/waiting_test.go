// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestWaiting(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "reports a loop that sleeps until it ends on a condition, and no loop that sleeps for a count or outside a test",
				give: "./eventually",
			},
			{name: "reports a loop that sleeps until a condition is true as EventuallyTrue", give: "./eventually-true"},
		}
		analyzeEach(t, tests)
	})
}
