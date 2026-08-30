// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Nil stops the test when got is not nil.
//
// A typed nil counts as nil. A (*T)(nil) held in an interface is not
// equal to a plain nil under ==, but it is nil for every purpose a
// test cares about.
func Nil(tb TB, got any, msg string) {
	tb.Helper()
	matcher.Nil(tb, matcher.Fatal, got, msg)
}

// NotNil stops the test when got is nil. A typed nil counts as nil;
// see [Nil].
func NotNil(tb TB, got any, msg string) {
	tb.Helper()
	matcher.NotNil(tb, matcher.Fatal, got, msg)
}
