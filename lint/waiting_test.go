// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestWaiting(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "reports a counted loop that sleeps between attempts as Eventually", give: "./eventually"},
			{name: "reports a loop that sleeps until a condition is true as EventuallyTrue", give: "./eventually-true"},
		}
		analyzeEach(t, tests)
	})
}
