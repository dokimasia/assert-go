// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// CloseTo records a failure when got is further than tolerance from want,
// comparing by absolute difference.
//
//	assert.CloseTo(t, elapsed.Seconds(), 1.0, 0.05, "the call took about a second")
//
// got is any numeric type, read as a float64. Values beyond 2^53 lose
// precision in that conversion, so compare large integers with [Equal]
// instead. A NaN on either side fails, since no tolerance contains it.
func CloseTo(tb assert.TB, got any, want, tolerance float64, msg string) {
	tb.Helper()
	matcher.CloseTo(tb, matcher.Soft, got, want, tolerance, msg)
}

// InRange records a failure when got falls outside the closed interval
// [low, high]. Both ends are included.
//
//	assert.InRange(t, port, 1024, 65535, "the port is unprivileged")
//
// got is any numeric type, with the precision limit [CloseTo]
// describes. A low above high always fails and says so.
func InRange(tb assert.TB, got any, low, high float64, msg string) {
	tb.Helper()
	matcher.InRange(tb, matcher.Soft, got, low, high, msg)
}
