// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestEdit(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "keeps the layout of a call and the name of its package in a fix", give: "./edits"},
		}
		analyzeEach(t, tests)
	})
}
