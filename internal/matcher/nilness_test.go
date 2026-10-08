// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestNilness runs the shared cases of Nil and NotNil.
func TestNilness(t *testing.T) {
	t.Parallel()

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.Nil(s, matcher.Fatal, got, msg)
			})
	})

	t.Run("NotNil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotNilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				matcher.NotNil(s, matcher.Fatal, got, msg)
			})
	})
}

// TestNilnessAllocs checks the allocation ceiling of a passing call of Nil
// and of NotNil.
func TestNilnessAllocs(t *testing.T) {
	checkAllocs(t, nilnessCases())
}

// BenchmarkNilness measures a passing call of Nil and of NotNil.
func BenchmarkNilness(b *testing.B) {
	benchAllocs(b, nilnessCases())
}

// nilnessCases returns a passing call of Nil on a nil pointer and of NotNil
// on a pointer, with its allocation ceiling.
func nilnessCases() []allocCase {
	var absent *int
	present := new(int)
	return []allocCase{
		{name: "Nil", call: func(seat matcher.Seat) { matcher.Nil(seat, matcher.Fatal, absent, allocContract) }},
		{name: "NotNil", call: func(seat matcher.Seat) { matcher.NotNil(seat, matcher.Fatal, present, allocContract) }},
	}
}
