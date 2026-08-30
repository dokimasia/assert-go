// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestText(t *testing.T) {
	t.Parallel()

	t.Run("HasPrefix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasPrefixCases(),
			func(s *matchertest.Seat, got, prefix any, msg string) {
				matcher.HasPrefix(s, matcher.Fatal, got, prefix.(string), msg)
			})
	})

	t.Run("HasSuffix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasSuffixCases(),
			func(s *matchertest.Seat, got, suffix any, msg string) {
				matcher.HasSuffix(s, matcher.Fatal, got, suffix.(string), msg)
			})
	})

	t.Run("Matches", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.MatchesCases(),
			func(s *matchertest.Seat, got, pattern any, msg string) {
				matcher.Matches(s, matcher.Fatal, got, pattern.(string), msg)
			})
	})
}
