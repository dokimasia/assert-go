// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// HasPrefix records a failure when got does not start with prefix.
//
// got is a string, a []byte, or any type defined over either.
//
// # Allocation contract
//
// A passing call on a string allocates nothing.
func HasPrefix(tb assert.TB, got any, prefix, msg string) {
	tb.Helper()
	matcher.HasPrefix(tb, matcher.Soft, got, prefix, msg)
}

// HasSuffix records a failure when got does not end with suffix. See
// [HasPrefix] for the types it reads.
//
// # Allocation contract
//
// A passing call on a string allocates nothing.
func HasSuffix(tb assert.TB, got any, suffix, msg string) {
	tb.Helper()
	matcher.HasSuffix(tb, matcher.Soft, got, suffix, msg)
}

// Matches records a failure when got does not match pattern, a pattern
// of the portable subset that every implementation of the standard
// reads the same way.
//
// The pattern matches anywhere in got. Anchor it with ^ and $ to
// require the whole value:
//
//	expect.Matches(t, id, `^[0-9a-f]{32}$`, "the id is a hex digest")
//
// $ matches at the end of the text only, \d, \w and \s are their ASCII
// classes, and . matches no line terminator. A pattern outside the
// subset, such as one with a backreference, a lookaround, a flag or \b,
// fails like any other assertion and does not panic, because a test with
// such a pattern has established nothing.
//
// # Allocation contract
//
// Matches compiles pattern on every call: a passing call of a pattern of
// a literal, a class and an anchor at each end allocates 62 times.
func Matches(tb assert.TB, got any, pattern, msg string) {
	tb.Helper()
	matcher.Matches(tb, matcher.Soft, got, pattern, msg)
}
