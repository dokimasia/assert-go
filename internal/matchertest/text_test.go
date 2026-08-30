// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestText(t *testing.T) {
	t.Parallel()

	t.Run("HasPrefixCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "HasPrefixCases", matchertest.HasPrefixCases(), 2)
	})

	t.Run("HasSuffixCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "HasSuffixCases", matchertest.HasSuffixCases(), 2)
	})

	t.Run("MatchesCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "MatchesCases", matchertest.MatchesCases(), 2)
	})
}
