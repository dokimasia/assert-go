// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestRejects(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "reports a check that another test failed as Rejects", give: "./rejects"},
		}
		analyzeEach(t, tests)
	})
}
