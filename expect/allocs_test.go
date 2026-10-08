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

// TestMaxAllocs does not run in parallel: its count covers the whole
// process.
func TestMaxAllocs(t *testing.T) {
	matchertest.RunMaxAllocs(t, func(s *matchertest.Seat, fn func(), ceiling uint64, msg string) {
		expect.MaxAllocs(s, fn, ceiling, msg)
	})
}

// TestMaxAllocsWithSetupAllocs does not run in parallel: its count covers the
// whole process.
func TestMaxAllocsWithSetupAllocs(t *testing.T) {
	matchertest.RunMaxAllocsWithSetup(t,
		func(s *matchertest.Seat, setup func() *[]byte, fn func(*[]byte), ceiling uint64, msg string) {
			expect.MaxAllocsWithSetup(s, setup, fn, ceiling, msg)
		})
}

// TestAllocsAllocs checks the allocation ceiling of a passing call of
// MaxAllocs and of MaxAllocsWithSetup.
func TestAllocsAllocs(t *testing.T) {
	alloctest.Check(t, allocsCases())
}

// BenchmarkAllocs measures a passing call of MaxAllocs and of
// MaxAllocsWithSetup.
func BenchmarkAllocs(b *testing.B) {
	for _, c := range allocsCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// allocsCases returns a passing call of MaxAllocs and of
// MaxAllocsWithSetup of functions that allocate nothing, with their
// allocation ceilings, measured.
func allocsCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "MaxAllocs", Call: func(tb assert.TB) { expect.MaxAllocs(tb, func() {}, 0, allocContract) }},
		{Name: "MaxAllocsWithSetup", Call: func(tb assert.TB) {
			expect.MaxAllocsWithSetup(tb, func() int { return 0 }, func(int) {}, 0, allocContract)
		}},
	}
}
