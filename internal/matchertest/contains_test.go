// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestContains(t *testing.T) {
	t.Parallel()

	t.Run("ContainsCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "ContainsCases", matchertest.ContainsCases(), 2)
	})

	t.Run("NotContainsCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "NotContainsCases", matchertest.NotContainsCases(), 2)
	})

	t.Run("ContainsInOrderCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "ContainsInOrderCases", matchertest.ContainsInOrderCases(), 2)
	})
}
