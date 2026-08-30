// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// True stops the test when cond is false.
//
// The failure carries msg alone, because a boolean that came out wrong
// has no detail worth printing. Where the values behind the condition
// matter, compare them with [Equal] instead and let the diff say so.
func True(tb TB, cond bool, msg string) {
	tb.Helper()
	matcher.True(tb, matcher.Fatal, cond, msg)
}

// False stops the test when cond is true. See [True].
func False(tb TB, cond bool, msg string) {
	tb.Helper()
	matcher.False(tb, matcher.Fatal, cond, msg)
}
