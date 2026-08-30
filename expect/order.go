// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Pairwise records a failure when any adjacent pair of items fails pred,
// which receives them in slice order as (earlier, later).
//
// It states an ordering without naming one: pass a less-than for
// ascending, an after for reverse-chronological, or any relation
// neighbours must satisfy.
//
//	assert.Pairwise(t, stamps, func(earlier, later time.Time) bool {
//	    return earlier.Before(later)
//	}, "the events are in chronological order")
//
// A slice of nought or one item passes, having no pair to break. The
// failure names the index and both values of the first break.
func Pairwise[T any](tb assert.TB, items []T, pred func(earlier, later T) bool, msg string) {
	tb.Helper()
	matcher.Pairwise(tb, matcher.Soft, items, pred, msg)
}
