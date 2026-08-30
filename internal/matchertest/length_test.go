// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestLength(t *testing.T) {
	t.Parallel()

	t.Run("LengthCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "LengthCases", matchertest.LengthCases(), 2)
	})

	t.Run("EmptyCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "EmptyCases", matchertest.EmptyCases(), 1)
	})

	t.Run("NotEmptyCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "NotEmptyCases", matchertest.NotEmptyCases(), 1)
	})
}
