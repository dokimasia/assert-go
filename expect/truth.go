// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// True records a failure when cond is false.
//
// The failure states msg alone, because a boolean has no detail to print.
// Where the values behind the condition matter, compare them with [Equal],
// whose failure shows their diff.
//
// # Allocation contract
//
// A passing call allocates nothing.
func True(tb assert.TB, cond bool, msg string) {
	tb.Helper()
	matcher.True(tb, matcher.Soft, cond, msg)
}

// False records a failure when cond is true. See [True].
//
// # Allocation contract
//
// A passing call allocates nothing.
func False(tb assert.TB, cond bool, msg string) {
	tb.Helper()
	matcher.False(tb, matcher.Soft, cond, msg)
}
