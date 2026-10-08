// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestNilness(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "suggests Nil for a comparison of a value whose nil is no interface", give: "./nil"},
			{name: "suggests NotNil for a check that such a value is present", give: "./not-nil"},
			{name: "suggests NoError for a comparison of an error with nil", give: "./err-absent"},
			{name: "suggests HasError for a check that an error is present", give: "./err-present"},
		}
		analyzeEach(t, tests)
	})
}
