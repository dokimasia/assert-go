// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestTruth runs the shared cases of True and False.
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

// TestTruthAllocs checks the allocation ceiling of a passing call of True
// and of False.
func TestTruthAllocs(t *testing.T) {
	checkAllocs(t, truthCases())
}

// BenchmarkTruth measures a passing call of True and of False.
func BenchmarkTruth(b *testing.B) {
	benchAllocs(b, truthCases())
}

// truthCases returns a passing call of True and of False, with its
// allocation ceiling.
func truthCases() []allocCase {
	return []allocCase{
		{name: "True", call: func(seat matcher.Seat) { matcher.True(seat, matcher.Fatal, true, allocContract) }},
		{name: "False", call: func(seat matcher.Seat) { matcher.False(seat, matcher.Fatal, false, allocContract) }},
	}
}
