// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestNilness(t *testing.T) {
	t.Parallel()

	t.Run("NilCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "NilCases", matchertest.NilCases(), 1)
	})

	t.Run("NotNilCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "NotNilCases", matchertest.NotNilCases(), 1)
	})
}
