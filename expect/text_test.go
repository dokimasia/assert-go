// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestText(t *testing.T) {
	t.Parallel()

	t.Run("HasPrefix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasPrefixCases(),
			func(s *matchertest.Seat, got, prefix any, msg string) {
				expect.HasPrefix(s, got, prefix.(string), msg)
			})
	})

	t.Run("HasSuffix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasSuffixCases(),
			func(s *matchertest.Seat, got, suffix any, msg string) {
				expect.HasSuffix(s, got, suffix.(string), msg)
			})
	})

	t.Run("Matches", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.MatchesCases(),
			func(s *matchertest.Seat, got, pattern any, msg string) {
				expect.Matches(s, got, pattern.(string), msg)
			})
	})
}
