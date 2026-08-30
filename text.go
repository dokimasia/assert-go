// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// HasPrefix stops the test when got does not start with prefix.
//
// got is a string, a []byte, or any type defined over either.
func HasPrefix(tb TB, got any, prefix, msg string) {
	tb.Helper()
	matcher.HasPrefix(tb, matcher.Fatal, got, prefix, msg)
}

// HasSuffix stops the test when got does not end with suffix. See
// [HasPrefix] for the types that answer.
func HasSuffix(tb TB, got any, suffix, msg string) {
	tb.Helper()
	matcher.HasSuffix(tb, matcher.Fatal, got, suffix, msg)
}

// Matches stops the test when got does not match the regular
// expression pattern.
//
// The pattern matches anywhere in got; anchor it with ^ and $ to
// require the whole value:
//
//	assert.Matches(t, id, `^[0-9a-f]{32}$`, "the id is a hex digest")
//
// A pattern that does not compile fails like any other assertion
// rather than panicking, because a test with a broken pattern has
// established nothing and should say so on the seat.
func Matches(tb TB, got any, pattern, msg string) {
	tb.Helper()
	matcher.Matches(tb, matcher.Fatal, got, pattern, msg)
}
