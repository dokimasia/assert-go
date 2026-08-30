// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("NoError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NoErrorCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.NoError(s, matchertest.AsError(got), msg)
			})
	})

	t.Run("HasError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.HasErrorCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.HasError(s, matchertest.AsError(got), msg)
			})
	})

	t.Run("ErrorIs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				assert.ErrorIs(s, matchertest.AsError(got), matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorIsNot", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsNotCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				assert.ErrorIsNot(s, matchertest.AsError(got), matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorAs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunErrorAs(t, func(s *matchertest.Seat, err error, msg string) *matchertest.TypedError {
			return assert.ErrorAs[*matchertest.TypedError](s, err, msg)
		})
	})
}
