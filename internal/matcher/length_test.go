// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestLength runs the shared cases of the length assertions.
func TestLength(t *testing.T) {
	t.Parallel()

	t.Run("Length", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.LengthCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				matcher.Length(s, matcher.Fatal, got, want.(int), msg)
			})
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.EmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.Empty(s, matcher.Fatal, got, msg)
			})
	})

	t.Run("NotEmpty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotEmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.NotEmpty(s, matcher.Fatal, got, msg)
			})
	})
}

// TestLengthAllocs checks the allocation ceiling of a passing call of each
// length assertion.
func TestLengthAllocs(t *testing.T) {
	checkAllocs(t, lengthCases())
}

// BenchmarkLength measures a passing call of each length assertion.
func BenchmarkLength(b *testing.B) {
	benchAllocs(b, lengthCases())
}

// lengthCases returns a passing call of each length assertion on a slice
// of ints, with its allocation ceiling, measured.
func lengthCases() []allocCase {
	items, none := []int{1, 2, 3}, []int{}
	return []allocCase{
		{name: "Length", call: func(seat matcher.Seat) {
			matcher.Length(seat, matcher.Fatal, items, 3, allocContract)
		}},
		{name: "Empty", allocs: 1, call: func(seat matcher.Seat) {
			matcher.Empty(seat, matcher.Fatal, none, allocContract)
		}},
		{name: "NotEmpty", allocs: 1, call: func(seat matcher.Seat) {
			matcher.NotEmpty(seat, matcher.Fatal, items, allocContract)
		}},
	}
}
