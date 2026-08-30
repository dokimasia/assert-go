// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matchertest"
)

func TestPanics(t *testing.T) {
	t.Parallel()

	t.Run("RunPanics", func(t *testing.T) {
		t.Parallel()

		matchertest.RunPanics(t, func(s *matchertest.Seat, fn func(), msg string) (recovered any) {
			panicked := true
			func() {
				defer func() { recovered = recover() }()
				fn()
				panicked = false
			}()
			if !panicked {
				s.Fatalf("%s: expected a panic, the function returned", msg)
			}
			return recovered
		})
	})

	t.Run("RunNotPanics", func(t *testing.T) {
		t.Parallel()

		matchertest.RunNotPanics(t, func(s *matchertest.Seat, fn func(), msg string) {
			defer func() {
				if r := recover(); r != nil {
					s.Fatalf("%s: unexpected panic: %v", msg, r)
				}
			}()
			fn()
		})
	})
}
