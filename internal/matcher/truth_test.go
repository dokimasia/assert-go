// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestTruth(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.TrueCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.True(s, matcher.Fatal, got.(bool), msg)
			})
	})

	t.Run("False", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.FalseCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.False(s, matcher.Fatal, got.(bool), msg)
			})
	})
}
