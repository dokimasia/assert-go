// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestEqual runs the shared cases of Equal and NotEqual.
func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.EqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				matcher.Equal(s, matcher.Fatal, got, want, msg)
			})
	})

	t.Run("NotEqual", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotEqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				matcher.NotEqual(s, matcher.Fatal, got, want, msg)
			})
	})
}

// TestEqualAllocs checks the allocation ceiling of a passing call of Equal
// and of NotEqual.
func TestEqualAllocs(t *testing.T) {
	checkAllocs(t, equalCases())
}

// BenchmarkEqual measures a passing call of Equal and of NotEqual.
func BenchmarkEqual(b *testing.B) {
	benchAllocs(b, equalCases())
}

// equalCases returns a passing call of Equal and of NotEqual on two ints,
// with its allocation ceiling.
func equalCases() []allocCase {
	return []allocCase{
		{name: "Equal", call: func(seat matcher.Seat) {
			matcher.Equal(seat, matcher.Fatal, 7, 7, allocContract)
		}},
		{name: "NotEqual", call: func(seat matcher.Seat) {
			matcher.NotEqual(seat, matcher.Fatal, 7, 8, allocContract)
		}},
	}
}
