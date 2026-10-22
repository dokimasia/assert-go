// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// checked and reason keep the results of the allocation case of
// AllocsChecked.
var (
	checked bool
	reason  string
)

// TestAllocsChecked checks the outcome of AllocsChecked in the running
// binary. The tests of the matcher check the kind of each build and run in
// child processes.
func TestAllocsChecked(t *testing.T) {
	t.Parallel()

	t.Run("AllocsChecked", func(t *testing.T) {
		t.Parallel()

		t.Run("names the build or the run that the matcher names", func(t *testing.T) {
			t.Parallel()

			if _, got := assert.AllocsChecked(); got != matcher.AllocationsUncounted() {
				t.Fatalf("AllocsChecked names %q, want %q", got, matcher.AllocationsUncounted())
			}
		})

		t.Run("reports a check exactly when it names no build or run", func(t *testing.T) {
			t.Parallel()

			if got, kind := assert.AllocsChecked(); got != (kind == "") {
				t.Fatalf("AllocsChecked = %v, %q, want true exactly for no kind", got, kind)
			}
		})
	})
}

// TestMaxAllocs does not run in parallel: its count covers the whole
// process.
func TestMaxAllocs(t *testing.T) {
	matchertest.RunMaxAllocs(t, func(s *matchertest.Seat, fn func(), ceiling uint64, msg string) {
		assert.MaxAllocs(s, fn, ceiling, msg)
	})
}

// TestMaxAllocsWithSetupAllocs does not run in parallel: its count covers the
// whole process.
func TestMaxAllocsWithSetupAllocs(t *testing.T) {
	matchertest.RunMaxAllocsWithSetup(t,
		func(s *matchertest.Seat, setup func() *[]byte, fn func(*[]byte), ceiling uint64, msg string) {
			assert.MaxAllocsWithSetup(s, setup, fn, ceiling, msg)
		})
}

// TestAllocsAllocs checks the allocation ceiling of a passing call of
// MaxAllocs and of MaxAllocsWithSetup, and of a call of AllocsChecked.
func TestAllocsAllocs(t *testing.T) {
	alloctest.Check(t, allocsCases())
}

// BenchmarkAllocs measures a passing call of MaxAllocs and of
// MaxAllocsWithSetup, and a call of AllocsChecked.
func BenchmarkAllocs(b *testing.B) {
	for _, c := range allocsCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// allocsCases returns a passing call of MaxAllocs and of MaxAllocsWithSetup
// of functions that allocate nothing, and a call of AllocsChecked, with their
// allocation ceilings.
func allocsCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "MaxAllocs", Call: func(tb assert.TB) { assert.MaxAllocs(tb, func() {}, 0, allocContract) }},
		{Name: "MaxAllocsWithSetup", Call: func(tb assert.TB) {
			assert.MaxAllocsWithSetup(tb, func() int { return 0 }, func(int) {}, 0, allocContract)
		}},
		{Name: "AllocsChecked", Call: func(assert.TB) { checked, reason = assert.AllocsChecked() }},
	}
}
