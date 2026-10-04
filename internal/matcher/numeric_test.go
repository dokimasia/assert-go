// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestNumeric runs the shared cases of the numeric assertions.
func TestNumeric(t *testing.T) {
	t.Parallel()

	t.Run("CloseTo", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.CloseToCases(),
			func(s *matchertest.Seat, got, want, tolerance any, msg string) {
				matcher.CloseTo(s, matcher.Fatal, got, want.(float64), tolerance.(float64), msg)
			})
	})

	t.Run("InRange", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.InRangeCases(),
			func(s *matchertest.Seat, got, low, high any, msg string) {
				matcher.InRange(s, matcher.Fatal, got, low.(float64), high.(float64), msg)
			})
	})
}

// TestNumericAllocs checks the allocation ceiling of a passing call of
// each numeric assertion.
func TestNumericAllocs(t *testing.T) {
	checkAllocs(t, numericCases())
}

// BenchmarkNumeric measures a passing call of each numeric assertion.
func BenchmarkNumeric(b *testing.B) {
	benchAllocs(b, numericCases())
}

// numericCases returns a passing call of each numeric assertion on a
// float64, with its allocation ceiling, measured.
func numericCases() []allocCase {
	reading := 1.05
	return []allocCase{
		{name: "CloseTo", call: func(seat matcher.Seat) {
			matcher.CloseTo(seat, matcher.Fatal, reading, 1, 0.1, allocContract)
		}},
		{name: "InRange", call: func(seat matcher.Seat) {
			matcher.InRange(seat, matcher.Fatal, reading, 0, 2, allocContract)
		}},
	}
}
