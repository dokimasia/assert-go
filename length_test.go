// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestLength(t *testing.T) {
	t.Parallel()

	t.Run("Length", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.LengthCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				assert.Length(s, got, want.(int), msg)
			})
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.EmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.Empty(s, got, msg)
			})
	})

	t.Run("NotEmpty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotEmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.NotEmpty(s, got, msg)
			})
	})
}
