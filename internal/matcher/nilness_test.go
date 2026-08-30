// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestNilness(t *testing.T) {
	t.Parallel()

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.Nil(s, matcher.Fatal, got, msg)
			})
	})

	t.Run("NotNil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotNilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.NotNil(s, matcher.Fatal, got, msg)
			})
	})
}
