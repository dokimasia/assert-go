// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestEqual(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "suggests Equal for ==, for reflect.DeepEqual over a type it compares alike, and for bytes.Equal",
				give: "./equal",
			},
			{
				name: "suggests NotEqual for !=, for reflect.DeepEqual over a type it compares alike, and for bytes.Equal",
				give: "./not-equal",
			},
		}
		analyzeEach(t, tests)
	})
}
