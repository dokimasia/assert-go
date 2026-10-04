// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package coverage_test

import "go.dokimi.dev/assert/internal/prop/coverage"

// The values past the members of each enumeration.
const (
	// invalidStage is the first value past the three stages.
	invalidStage coverage.Stage = 3
	// invalidVerdict is the first value past the four verdicts.
	invalidVerdict coverage.Verdict = 4
)
