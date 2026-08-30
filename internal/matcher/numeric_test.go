// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestNumeric(t *testing.T) {
	t.Parallel()

	t.Run("CloseTo", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.CloseToCases(),
			func(s *matchertest.Seat, got, want, tolerance any, msg string) {
				matcher.CloseTo(s, matcher.Fatal, got, want.(float64), tolerance.(float64), msg)
			})
	})

	t.Run("InRange", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.InRangeCases(),
			func(s *matchertest.Seat, got, low, high any, msg string) {
				matcher.InRange(s, matcher.Fatal, got, low.(float64), high.(float64), msg)
			})
	})
}
