// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestOrder(t *testing.T) {
	t.Parallel()

	t.Run("Pairwise", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPairwise(t, func(s *matchertest.Seat, items []int,
			pred func(earlier, later int) bool, msg string,
		) {
			matcher.Pairwise(s, matcher.Fatal, items, pred, msg)
		})
	})
}
