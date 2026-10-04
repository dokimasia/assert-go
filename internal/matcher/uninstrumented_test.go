// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !race && !msan && !asan

package matcher_test

import (
	"runtime/debug"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

// TestUninstrumented checks a build without the race detector, msan or
// asan, which counts the allocations of the code under test.
func TestUninstrumented(t *testing.T) {
	t.Parallel()

	t.Run("AllocationsCounted", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true unless the build's -gcflags turn off optimisation", func(t *testing.T) {
			t.Parallel()

			info, _ := debug.ReadBuildInfo()
			if got, want := matcher.AllocationsCounted(), !matcher.OptimisationsOff(info); got != want {
				t.Fatalf("AllocationsCounted = %v, want %v in a build without instrumentation", got, want)
			}
		})
	})
}
