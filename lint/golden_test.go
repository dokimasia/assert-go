// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestGolden(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "reports a comparison with a file under testdata/golden, and the flag -update of a test file",
				give: "./golden-match",
			},
			{name: "reports a comparison with a file at another path as golden.MatchAt", give: "./golden-match-at"},
			{name: "reports a comparison of a file with a constant as files.HasContent", give: "./has-content"},
		}
		analyzeEach(t, tests)
	})
}
