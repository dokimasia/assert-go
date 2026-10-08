// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestEqual runs the shared cases of Equal and NotEqual.
func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.EqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				assert.Equal(s, got, want, msg)
			})
	})

	t.Run("NotEqual", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotEqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				assert.NotEqual(s, got, want, msg)
			})
	})
}

// TestEqualAllocs checks the allocation ceiling of a passing call of Equal
// and of NotEqual.
func TestEqualAllocs(t *testing.T) {
	alloctest.Check(t, equalCases())
}

// BenchmarkEqual measures a passing call of Equal and of NotEqual.
func BenchmarkEqual(b *testing.B) {
	for _, c := range equalCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// equalCases returns a passing call of Equal and of NotEqual on two ints,
// with its allocation ceiling, measured.
func equalCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "Equal", Call: func(tb assert.TB) { assert.Equal(tb, 7, 7, allocContract) }},
		{Name: "NotEqual", Call: func(tb assert.TB) { assert.NotEqual(tb, 7, 8, allocContract) }},
	}
}
