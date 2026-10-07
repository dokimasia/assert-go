// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestMeasurement(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "reports a check of testing.AllocsPerRun as MaxAllocs", give: "./max-allocs"},
			{
				name: "reports a check of the allocations of a setup and a call as MaxAllocs",
				give: "./max-allocs-with-setup",
			},
			{name: "reports a check of runtime.NumGoroutine as NoGoroutineLeaks", give: "./no-task-leaks"},
			{
				name: "reports a check of the allocations of a benchmark as a contract's MaxAllocs",
				give: "./bench-max-allocs",
			},
			{name: "reports a check of the bytes of a benchmark as a contract's MaxBytes", give: "./bench-max-bytes"},
			{name: "reports a check of the time of a benchmark as a contract's MaxMean", give: "./bench-max-mean"},
		}
		analyzeEach(t, tests)
	})
}
