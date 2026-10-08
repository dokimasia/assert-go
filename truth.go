// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// True stops the test when cond is false.
//
// The failure states msg alone, because a boolean has no detail to print.
// Where the values behind the condition matter, compare them with [Equal],
// whose failure shows their diff.
//
// # Allocation contract
//
// A passing call allocates nothing.
func True(tb TB, cond bool, msg string) {
	tb.Helper()
	matcher.True(tb, matcher.Fatal, cond, msg)
}

// False stops the test when cond is true. See [True].
//
// # Allocation contract
//
// A passing call allocates nothing.
func False(tb TB, cond bool, msg string) {
	tb.Helper()
	matcher.False(tb, matcher.Fatal, cond, msg)
}
