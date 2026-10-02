// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// TestCoverage checks a coverage vector: the verdict on one requirement at
// one check.
func TestCoverage(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns nil for a vector that states the verdict",
				give: `{"counted":27,"valid":100,"share":0.1,"last":false,"exact":false,"verdict":"met"}`,
			},
			{
				name: "returns nil for the verdict of the exact share at the last check",
				give: `{"counted":9100,"valid":100000,"share":0.1,"last":true,"exact":true,"verdict":"met"}`,
			},
			{
				name: "returns an error for a vector that is no JSON object",
				give: `[]`,
				want: "cannot unmarshal array",
			},
			{
				name: "returns an error for another verdict",
				give: `{"counted":27,"valid":100,"share":0.1,"last":false,"exact":false,"verdict":"refuted"}`,
				want: "the verdict is met, want refuted",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Coverage, tt.give), tt.want)
			})
		}
	})
}
