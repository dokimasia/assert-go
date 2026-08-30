// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// CloseTo stops the test when got is further than tolerance from want,
// comparing by absolute difference.
//
//	assert.CloseTo(t, elapsed.Seconds(), 1.0, 0.05, "the call took about a second")
//
// got is any numeric type, read as a float64. Values beyond 2^53 lose
// precision in that conversion, so compare large integers with [Equal]
// instead. A NaN on either side fails, since no tolerance contains it.
func CloseTo(tb TB, got any, want, tolerance float64, msg string) {
	tb.Helper()
	matcher.CloseTo(tb, matcher.Fatal, got, want, tolerance, msg)
}

// InRange stops the test when got falls outside the closed interval
// [low, high]. Both ends are included.
//
//	assert.InRange(t, port, 1024, 65535, "the port is unprivileged")
//
// got is any numeric type, with the precision limit [CloseTo]
// describes. A low above high always fails and says so.
func InRange(tb TB, got any, low, high float64, msg string) {
	tb.Helper()
	matcher.InRange(tb, matcher.Fatal, got, low, high, msg)
}
