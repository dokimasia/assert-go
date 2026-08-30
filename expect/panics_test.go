// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestPanics(t *testing.T) {
	t.Parallel()

	t.Run("Panics", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPanics(t, func(s *matchertest.Seat, fn func(), msg string) any {
			return expect.Panics(s, fn, msg)
		})
	})

	t.Run("NotPanics", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNotPanics(t, func(s *matchertest.Seat, fn func(), msg string) {
			expect.NotPanics(s, fn, msg)
		})
	})
}
