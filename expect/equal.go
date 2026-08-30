// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Equal compares got against want and records a failure and lets the test continue when they differ.
// It reports nothing and returns when they are equal.
//
// The failure is msg, then a diff labelled -want +got. opts relax the
// comparison for this call alone.
//
//	assert.Equal(t, store.Get(ctx, id), item, "Get returns the stored item")
//
// Comparison is structural and reaches every field, including
// unexported ones. A nil map or slice does not equal an empty one;
// pass [EquateEmpty] where that difference does not matter. Floats
// compare exactly and NaN does not equal itself.
//
// got and want share a type parameter, so a mismatch is a compile
// error rather than a failure at run time.
func Equal[T any](tb assert.TB, got, want T, msg string, opts ...Option) {
	tb.Helper()
	matcher.Equal(tb, matcher.Soft, got, want, msg, opts...)
}

// NotEqual compares got against want and records a failure and lets the test continue when they are
// equal. It reports nothing and returns when they differ.
//
// The failure is msg, then got. There is no diff to show: the two
// values matched, so printing one says everything the reader needs.
// opts relax the comparison for this call alone, under the same rules
// [Equal] describes.
//
//	assert.NotEqual(t, token, previous, "Refresh issues a new token")
func NotEqual[T any](tb assert.TB, got, want T, msg string, opts ...Option) {
	tb.Helper()
	matcher.NotEqual(tb, matcher.Soft, got, want, msg, opts...)
}
