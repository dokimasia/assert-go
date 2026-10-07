// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestTruth(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "suggests a True of each operand of a conjunction, and reports an if check as True",
				give: "./true",
			},
			{name: "reports an if check of a condition that must be false as False", give: "./false"},
		}
		analyzeEach(t, tests)
	})
}
