// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestTruth(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.TrueCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.True(s, got.(bool), msg)
			})
	})

	t.Run("False", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.FalseCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.False(s, got.(bool), msg)
			})
	})
}
