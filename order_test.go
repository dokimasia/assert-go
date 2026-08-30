// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestOrder(t *testing.T) {
	t.Parallel()

	t.Run("Pairwise", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPairwise(t, func(s *matchertest.Seat, items []int,
			pred func(earlier, later int) bool, msg string,
		) {
			assert.Pairwise(s, items, pred, msg)
		})
	})
}
