// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestSkip(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		analyzeEach(t, []fixture{{
			name: "leaves out the reports that an annotation covers, and reports an annotation that leaves out none",
			give: "./lint-skip",
		}})
	})
}
