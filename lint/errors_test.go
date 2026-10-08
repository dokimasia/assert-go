// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestErrors(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "suggests ErrorIs for errors.Is, and reports an error that == compares with a sentinel",
				give: "./err-is",
			},
			{name: "suggests ErrorIsNot for a check that an error does not match", give: "./err-is-not"},
			{
				name: "suggests ErrorAs for a statement of assert whose target's type the file writes",
				give: "./err-as/...",
			},
		}
		analyzeEach(t, tests)
	})
}
