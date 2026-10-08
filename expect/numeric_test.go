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

// TestNumeric runs the shared cases of the numeric assertions.
func TestNumeric(t *testing.T) {
	t.Parallel()

	t.Run("CloseTo", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.CloseToCases(),
			func(s *matchertest.Seat, got, want, tolerance any, msg string) {
				expect.CloseTo(s, got, want.(float64), tolerance.(float64), msg)
			})
	})

	t.Run("InRange", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.InRangeCases(),
			func(s *matchertest.Seat, got, low, high any, msg string) {
				expect.InRange(s, got, low.(float64), high.(float64), msg)
			})
	})
}

// TestNumericAllocs checks the allocation ceiling of a passing call of
// each numeric assertion.
func TestNumericAllocs(t *testing.T) {
	alloctest.Check(t, numericCases())
}

// BenchmarkNumeric measures a passing call of each numeric assertion.
func BenchmarkNumeric(b *testing.B) {
	for _, c := range numericCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// numericCases returns a passing call of each numeric assertion on a
// float64, with its allocation ceiling.
func numericCases() []alloctest.Case {
	reading := 1.05
	return []alloctest.Case{
		{Name: "CloseTo", Call: func(tb assert.TB) { expect.CloseTo(tb, reading, 1, 0.1, allocContract) }},
		{Name: "InRange", Call: func(tb assert.TB) { expect.InRange(tb, reading, 0, 2, allocContract) }},
	}
}
