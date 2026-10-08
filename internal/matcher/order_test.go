// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
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
			matcher.Pairwise(s, matcher.Fatal, items, pred, msg)
		})

		t.Run("writes the record of a passing call", func(t *testing.T) {
			t.Parallel()
			ascending := func(earlier, later int) bool { return earlier < later }
			checkPassRecord(t, "pairwise", func(seat matcher.Seat) {
				matcher.Pairwise(seat, matcher.Fatal, []int{1, 2}, ascending, allocContract)
			})
		})
	})
}

// TestOrderAllocs checks the allocation ceiling of a passing call of
// Pairwise.
func TestOrderAllocs(t *testing.T) {
	checkAllocs(t, orderCases())
}

// BenchmarkOrder measures a passing call of Pairwise.
func BenchmarkOrder(b *testing.B) {
	benchAllocs(b, orderCases())
}

// orderCases returns a passing call of Pairwise on three ascending ints,
// with its allocation ceiling, measured.
func orderCases() []allocCase {
	items := []int{1, 2, 3}
	ascending := func(earlier, later int) bool { return earlier < later }
	return []allocCase{
		{name: "Pairwise", call: func(seat matcher.Seat) {
			matcher.Pairwise(seat, matcher.Fatal, items, ascending, allocContract)
		}},
	}
}
