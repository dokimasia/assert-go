// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestPanics(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "reports a deferred check that recover returns a value as Panics", give: "./throws"},
			{name: "reports a deferred check that recover returns nil as NotPanics", give: "./not-throws"},
		}
		analyzeEach(t, tests)
	})
}
