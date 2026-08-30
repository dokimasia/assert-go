// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestContains(t *testing.T) {
	t.Parallel()

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				assert.Contains(s, haystack, needle, msg)
			})
	})

	t.Run("NotContains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				assert.NotContains(s, haystack, needle, msg)
			})
	})

	t.Run("ContainsInOrder", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsInOrderCases(),
			func(s *matchertest.Seat, got, needles any, msg string) {
				assert.ContainsInOrder(s, got, needles.([]string), msg)
			})
	})
}
