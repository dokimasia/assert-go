// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Length stops the test when got does not hold want items.
//
// It answers for an array, slice, map, channel or string. Passing
// anything else fails, naming the type, rather than panicking.
func Length(tb TB, got any, want int, msg string) {
	tb.Helper()
	matcher.Length(tb, matcher.Fatal, got, want, msg)
}

// Empty stops the test when got holds anything. See [Length] for the
// types that answer.
func Empty(tb TB, got any, msg string) {
	tb.Helper()
	matcher.Empty(tb, matcher.Fatal, got, msg)
}

// NotEmpty stops the test when got holds nothing. See [Length] for the
// types that answer.
func NotEmpty(tb TB, got any, msg string) {
	tb.Helper()
	matcher.NotEmpty(tb, matcher.Fatal, got, msg)
}
