// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestCheck(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "reads a call of an assertion of either surface and an if statement of one failure of a test",
				give: "./checks/...",
			},
		}
		analyzeEach(t, tests)
	})
}
