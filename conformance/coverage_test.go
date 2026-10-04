// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// TestCoverage checks a coverage vector: the verdict on one requirement at
// one check. Written with testing rather than with this library, because a
// verdict is not written with the subject.
func TestCoverage(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
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
				name:       "returns a fault at the verdict for another verdict",
				give:       `{"counted":27,"valid":100,"share":0.1,"last":false,"exact":false,"verdict":"refuted"}`,
				wantPath:   inVector(fault.Field("verdict")),
				wantReason: "the verdict is met, want refuted",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Coverage, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}
