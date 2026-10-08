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

// TestOrder runs the shared cases of Pairwise.
func TestOrder(t *testing.T) {
	t.Parallel()

	t.Run("Pairwise", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPairwise(t, func(s *matchertest.Seat, items []int,
			pred func(earlier, later int) bool, msg string,
		) {
			expect.Pairwise(s, items, pred, msg)
		})
	})
}

// TestOrderAllocs checks the allocation ceiling of a passing call of
// Pairwise.
func TestOrderAllocs(t *testing.T) {
	alloctest.Check(t, orderCases())
}

// BenchmarkOrder measures a passing call of Pairwise.
func BenchmarkOrder(b *testing.B) {
	for _, c := range orderCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// orderCases returns a passing call of Pairwise on three ascending ints,
// with its allocation ceiling, measured.
func orderCases() []alloctest.Case {
	items := []int{1, 2, 3}
	ascending := func(earlier, later int) bool { return earlier < later }
	return []alloctest.Case{
		{Name: "Pairwise", Call: func(tb assert.TB) { expect.Pairwise(tb, items, ascending, allocContract) }},
	}
}
