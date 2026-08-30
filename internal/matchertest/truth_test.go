// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestTruth(t *testing.T) {
	t.Parallel()

	t.Run("TrueCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "TrueCases", matchertest.TrueCases(), 1)
	})

	t.Run("FalseCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "FalseCases", matchertest.FalseCases(), 1)
	})
}
