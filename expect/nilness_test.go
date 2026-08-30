// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestNilness(t *testing.T) {
	t.Parallel()

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.Nil(s, got, msg)
			})
	})

	t.Run("NotNil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotNilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.NotNil(s, got, msg)
			})
	})
}
