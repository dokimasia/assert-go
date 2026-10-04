// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

//go:build race || msan || asan

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

// TestInstrumented checks a build with the race detector, msan or asan,
// which allocates on the code's behalf.
func TestInstrumented(t *testing.T) {
	t.Parallel()

	t.Run("AllocationsCounted", func(t *testing.T) {
		t.Parallel()

		t.Run("reports false", func(t *testing.T) {
			t.Parallel()

			if matcher.AllocationsCounted() {
				t.Fatal("AllocationsCounted = true, want false in a build with instrumentation")
			}
		})
	})
}
