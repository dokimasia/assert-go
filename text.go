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
// [HasPrefix] for the types it reads.
func HasSuffix(tb TB, got any, suffix, msg string) {
	tb.Helper()
	matcher.HasSuffix(tb, matcher.Fatal, got, suffix, msg)
}

// Matches stops the test when got does not match pattern, a pattern of
// the portable subset that every implementation of the standard reads
// the same way.
//
// The pattern matches anywhere in got. Anchor it with ^ and $ to
// require the whole value:
//
//	assert.Matches(t, id, `^[0-9a-f]{32}$`, "the id is a hex digest")
//
// $ matches at the end of the text only, \d, \w and \s are their ASCII
// classes, and . matches no line terminator. A pattern outside the
// subset, such as one with a backreference, a lookaround, a flag or \b,
// fails like any other assertion and does not panic, because a test with
// such a pattern has established nothing.
func Matches(tb TB, got any, pattern, msg string) {
	tb.Helper()
	matcher.Matches(tb, matcher.Fatal, got, pattern, msg)
}
