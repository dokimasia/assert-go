// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestText(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "suggests HasPrefix for strings.HasPrefix and bytes.HasPrefix", give: "./has-prefix"},
			{name: "suggests HasSuffix for strings.HasSuffix and bytes.HasSuffix", give: "./has-suffix"},
			{name: "reports a match of a regular expression as Matches without a fix", give: "./matches"},
		}
		analyzeEach(t, tests)
	})
}
