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

// TestNilness runs the shared cases of Nil and NotNil.
func TestNilness(t *testing.T) {
	t.Parallel()

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.Nil(s, got, msg)
			})
	})

	t.Run("NotNil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotNilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				expect.NotNil(s, got, msg)
			})
	})
}

// TestNilnessAllocs checks the allocation ceiling of a passing call of Nil
// and of NotNil.
func TestNilnessAllocs(t *testing.T) {
	alloctest.Check(t, nilnessCases())
}

// BenchmarkNilness measures a passing call of Nil and of NotNil.
func BenchmarkNilness(b *testing.B) {
	for _, c := range nilnessCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// nilnessCases returns a passing call of Nil on a nil pointer and of NotNil
// on a pointer, with its allocation ceiling, measured.
func nilnessCases() []alloctest.Case {
	var absent *int
	present := new(int)
	return []alloctest.Case{
		{Name: "Nil", Call: func(tb assert.TB) { expect.Nil(tb, absent, allocContract) }},
		{Name: "NotNil", Call: func(tb assert.TB) { expect.NotNil(tb, present, allocContract) }},
	}
}
