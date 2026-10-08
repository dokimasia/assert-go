// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestContains(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "suggests Contains for a search of text or of a slice, and for comparisons of one variable",
				give: "./contains",
			},
			{name: "suggests NotContains for a search that must find nothing", give: "./not-contains"},
			{name: "reports an order of two indices in one text as ContainsInOrder", give: "./contains-in-order"},
			{
				name: "reports an equality of a slice that a statement before it sorts as Permutation",
				give: "./permutation",
			},
		}
		analyzeEach(t, tests)
	})
}
