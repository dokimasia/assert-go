// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Length records a failure when got does not hold want items.
//
// It answers for an array, slice, map, channel or string. Passing
// anything else fails, naming the type, rather than panicking.
func Length(tb assert.TB, got any, want int, msg string) {
	tb.Helper()
	matcher.Length(tb, matcher.Soft, got, want, msg)
}

// Empty records a failure when got holds anything. See [Length] for the
// types that answer.
func Empty(tb assert.TB, got any, msg string) {
	tb.Helper()
	matcher.Empty(tb, matcher.Soft, got, msg)
}

// NotEmpty records a failure when got holds nothing. See [Length] for the
// types that answer.
func NotEmpty(tb assert.TB, got any, msg string) {
	tb.Helper()
	matcher.NotEmpty(tb, matcher.Soft, got, msg)
}
