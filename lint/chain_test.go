// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestChain(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "suggests That for consecutive assertions of one surface on one variable that no other fix edits",
				give: "./chain",
			},
		}
		analyzeEach(t, tests)
	})
}
