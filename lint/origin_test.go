// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestOrigin(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{
				name: "follows a value to the call that assigns or declares it, receives its address or converts it, " +
					"four steps deep",
				give: "./origins",
			},
		}
		analyzeEach(t, tests)
	})
}
