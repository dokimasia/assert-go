// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestLength(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "suggests Length for a count of a slice, an array, a map or a channel", give: "./length"},
			{name: "suggests Empty for a length of 0, a string's included", give: "./empty"},
			{name: "suggests NotEmpty for a length above 0, a string's included", give: "./not-empty"},
		}
		analyzeEach(t, tests)
	})
}
