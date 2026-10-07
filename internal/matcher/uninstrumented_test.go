// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !race && !msan && !asan

package matcher_test

import (
	"flag"
	"os"
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

		t.Run("reports true without optimisation off, mutation instrumentation or a test log",
			func(t *testing.T) {
				t.Parallel()

				info, _ := debug.ReadBuildInfo()
				_, mutated := os.LookupEnv(instrumentedVariable)
				logged := flag.Lookup("test.testlogfile").Value.String() != ""
				want := !matcher.OptimisationsOff(info) && !mutated && !logged
				if got := matcher.AllocationsCounted(); got != want {
					t.Fatalf("AllocationsCounted = %v, want %v in a build without instrumentation", got, want)
				}
			})
	})
}
