// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestOrder(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "reports a check that values are sorted as Pairwise without a fix", give: "./pairwise"},
		}
		analyzeEach(t, tests)
	})
}
