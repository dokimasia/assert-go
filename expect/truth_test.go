// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestTruth runs the shared cases of True and False.
func TestTruth(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.TrueCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.True(s, got.(bool), msg)
			})
	})

	t.Run("False", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.FalseCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.False(s, got.(bool), msg)
			})
	})
}

// TestTruthAllocs checks the allocation ceiling of a passing call of True
// and of False.
func TestTruthAllocs(t *testing.T) {
	alloctest.Check(t, truthCases())
}

// BenchmarkTruth measures a passing call of True and of False.
func BenchmarkTruth(b *testing.B) {
	for _, c := range truthCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// truthCases returns a passing call of True and of False, with its
// allocation ceiling, measured.
func truthCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "True", Call: func(tb assert.TB) { expect.True(tb, true, allocContract) }},
		{Name: "False", Call: func(tb assert.TB) { expect.False(tb, false, allocContract) }},
	}
}
