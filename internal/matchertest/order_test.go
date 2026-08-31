// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestOrder(t *testing.T) {
	t.Parallel()

	t.Run("RunPairwise", func(t *testing.T) {
		t.Parallel()

		// A correct pairwise implementation must satisfy every case
		// the suite states. Driving one here proves the suite is
		// satisfiable, which a suite nobody can pass is not.
		matchertest.RunPairwise(t, func(s *matchertest.Seat, items []int,
			pred func(earlier, later int) bool, msg string,
		) {
			for i := 1; i < len(items); i++ {
				if !pred(items[i-1], items[i]) {
					s.Report(matcher.Failure{
						Assertion: "pairwise", Contract: msg,
						Detail: map[string]any{"index": i - 1, "first": items[i-1], "second": items[i]},
					}, true)
					return
				}
			}
		})
	})
}
