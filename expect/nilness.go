// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Nil records a failure when got is not nil.
//
// A typed nil counts as nil. An interface that contains a (*T)(nil) is not
// equal to a plain nil under ==, and Nil passes on it.
//
// # Allocation contract
//
// A passing call on a nil pointer allocates nothing.
func Nil(tb assert.TB, got any, msg string) {
	tb.Helper()
	matcher.Nil(tb, matcher.Soft, got, msg)
}

// NotNil records a failure when got is nil. A typed nil counts as nil, as
// [Nil] describes.
//
// # Allocation contract
//
// A passing call on a pointer allocates nothing.
func NotNil(tb assert.TB, got any, msg string) {
	tb.Helper()
	matcher.NotNil(tb, matcher.Soft, got, msg)
}
