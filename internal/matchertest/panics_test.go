// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
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
				s.Report(matcher.Failure{Assertion: "throws", Contract: msg}, true)
			}
			return recovered
		})
	})

	t.Run("RunNotPanics", func(t *testing.T) {
		t.Parallel()

		matchertest.RunNotPanics(t, func(s *matchertest.Seat, fn func(), msg string) {
			defer func() {
				if r := recover(); r != nil {
					s.Report(matcher.Failure{
						Assertion: "not-throws", Contract: msg,
						Detail: map[string]any{"got": r},
					}, true)
				}
			}()
			fn()
		})
	})
}
