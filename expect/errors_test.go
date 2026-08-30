// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("NoError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NoErrorCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.NoError(s, matchertest.AsError(got), msg)
			})
	})

	t.Run("HasError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.HasErrorCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.HasError(s, matchertest.AsError(got), msg)
			})
	})

	t.Run("ErrorIs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				expect.ErrorIs(s, matchertest.AsError(got), matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorIsNot", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsNotCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				expect.ErrorIsNot(s, matchertest.AsError(got), matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorAs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunErrorAs(t, func(s *matchertest.Seat, err error, msg string) *matchertest.TypedError {
			return expect.ErrorAs[*matchertest.TypedError](s, err, msg)
		})
	})
}
