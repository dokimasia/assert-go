// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.EqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				expect.Equal(s, got, want, msg)
			})
	})

	t.Run("NotEqual", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotEqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				expect.NotEqual(s, got, want, msg)
			})
	})
}
