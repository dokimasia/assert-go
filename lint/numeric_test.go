// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestNumeric(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "suggests CloseTo for a difference within a tolerance under <=", give: "./close-to"},
			{
				name: "suggests InRange for two orders and for an order whose bounds a float64 states",
				give: "./in-range",
			},
		}
		analyzeEach(t, tests)
	})
}
