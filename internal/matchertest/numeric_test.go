// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestNumeric(t *testing.T) {
	t.Parallel()

	t.Run("CloseToCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "CloseToCases", matchertest.CloseToCases(), 3)
	})

	t.Run("InRangeCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "InRangeCases", matchertest.InRangeCases(), 3)
	})
}
