// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// NoError records a failure when err is not nil, naming the error.
//
// Use it for an operation that must succeed. Where the failure is the
// subject, reach for [ErrorIs].
func NoError(tb assert.TB, err error, msg string) {
	tb.Helper()
	matcher.NoError(tb, matcher.Soft, err, msg)
}

// HasError records a failure when err is nil.
//
// It asks only that something failed. Where which failure matters, and
// it usually does, [ErrorIs] says so and this does not.
func HasError(tb assert.TB, err error, msg string) {
	tb.Helper()
	matcher.HasError(tb, matcher.Soft, err, msg)
}

// ErrorIs records a failure when err does not match target under
// [errors.Is], which walks the chain of wrapped causes. A sentinel
// matches however deeply it was wrapped on the way up.
//
//	assert.ErrorIs(t, err, store.ErrNotFound, "Get reports a missing key")
func ErrorIs(tb assert.TB, err, target error, msg string) {
	tb.Helper()
	matcher.ErrorIs(tb, matcher.Soft, err, target, msg)
}

// ErrorIsNot records a failure when err matches target under
// [errors.Is]. Use it to hold two sentinels apart, where one matching
// the other would leave a caller unable to tell the cases apart.
func ErrorIsNot(tb assert.TB, err, target error, msg string) {
	tb.Helper()
	matcher.ErrorIsNot(tb, matcher.Soft, err, target, msg)
}

// ErrorAs finds the first error of type T in err's chain and returns
// it, stopping the test when the chain holds none.
//
//	notFound := assert.ErrorAs[*store.NotFoundError](t, err, "Get reports a missing key")
//	assert.Equal(t, notFound.Key, "absent", "and names the key")
//
// On failure the test stops, so the returned value is only ever read
// after a match. Under the recording surface it is the zero T, which
// keeps a chained read from dereferencing nil.
func ErrorAs[T any](tb assert.TB, err error, msg string) T {
	tb.Helper()
	return matcher.ErrorAs[T](tb, matcher.Soft, err, msg)
}
