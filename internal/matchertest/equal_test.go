// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("EqualCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "EqualCases", matchertest.EqualCases(), 2)
	})

	t.Run("NotEqualCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "NotEqualCases", matchertest.NotEqualCases(), 2)
	})
}
