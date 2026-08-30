// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Nil records a failure when got is not nil.
//
// A typed nil counts as nil. A (*T)(nil) held in an interface is not
// equal to a plain nil under ==, but it is nil for every purpose a
// test cares about.
func Nil(tb assert.TB, got any, msg string) {
	tb.Helper()
	matcher.Nil(tb, matcher.Soft, got, msg)
}

// NotNil records a failure when got is nil. A typed nil counts as nil;
// see [Nil].
func NotNil(tb assert.TB, got any, msg string) {
	tb.Helper()
	matcher.NotNil(tb, matcher.Soft, got, msg)
}
