// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestLength runs the shared cases of the length assertions.
func TestLength(t *testing.T) {
	t.Parallel()

	t.Run("Length", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.LengthCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				assert.Length(s, got, want.(int), msg)
			})
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.EmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.Empty(s, got, msg)
			})
	})

	t.Run("NotEmpty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotEmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.NotEmpty(s, got, msg)
			})
	})
}

// TestLengthAllocs checks the allocation ceiling of a passing call of each
// length assertion.
func TestLengthAllocs(t *testing.T) {
	alloctest.Check(t, lengthCases())
}

// BenchmarkLength measures a passing call of each length assertion.
func BenchmarkLength(b *testing.B) {
	for _, c := range lengthCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// lengthCases returns a passing call of each length assertion on a slice
// of ints, with its allocation ceiling, measured.
func lengthCases() []alloctest.Case {
	items, none := []int{1, 2, 3}, []int{}
	return []alloctest.Case{
		{Name: "Length", Call: func(tb assert.TB) { assert.Length(tb, items, 3, allocContract) }},
		{Name: "Empty", Allocs: 1, Call: func(tb assert.TB) { assert.Empty(tb, none, allocContract) }},
		{Name: "NotEmpty", Allocs: 1, Call: func(tb assert.TB) { assert.NotEmpty(tb, items, allocContract) }},
	}
}
