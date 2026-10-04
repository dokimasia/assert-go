// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Equal compares got against want and stops the test when they differ.
// It reports nothing and returns when they are equal.
//
// The failure is msg, then a diff labelled -want +got. opts relax the
// comparison for this call alone.
//
//	assert.Equal(t, store.Get(ctx, id), item, "Get returns the stored item")
//
// Comparison is structural and includes every field, unexported ones
// too. A nil map or slice does not equal an empty one. Pass [EquateEmpty]
// where that difference does not matter. Floats compare by IEEE 754
// equality, so NaN does not equal itself and negative zero equals zero.
//
// got and want share a type parameter, so a mismatch is a compile
// error rather than a failure at run time.
//
// # Allocation contract
//
// A passing call on two ints allocates 24 times, in go-cmp and in the
// comparison options that each call builds.
func Equal[T any](tb TB, got, want T, msg string, opts ...Option) {
	tb.Helper()
	matcher.Equal(tb, matcher.Fatal, got, want, msg, opts...)
}

// NotEqual compares got against want and stops the test when they are
// equal. It reports nothing and returns when they differ.
//
// The failure is msg, then got. The two values are equal, so got alone
// shows both. opts relax the comparison for this call alone, under the
// same rules [Equal] describes.
//
//	assert.NotEqual(t, token, previous, "Refresh issues a new token")
//
// # Allocation contract
//
// A passing call on two ints allocates 24 times, as [Equal] does.
func NotEqual[T any](tb TB, got, want T, msg string, opts ...Option) {
	tb.Helper()
	matcher.NotEqual(tb, matcher.Fatal, got, want, msg, opts...)
}
