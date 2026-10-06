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
// subject, use [ErrorIs].
//
// # Allocation contract
//
// A passing call allocates nothing.
func NoError(tb assert.TB, err error, msg string) {
	tb.Helper()
	matcher.NoError(tb, matcher.Soft, err, msg)
}

// HasError records a failure when err is nil.
//
// It checks only that something failed. Where the kind of failure
// matters, use [ErrorIs], which checks the kind.
//
// # Allocation contract
//
// A passing call allocates nothing.
func HasError(tb assert.TB, err error, msg string) {
	tb.Helper()
	matcher.HasError(tb, matcher.Soft, err, msg)
}

// ErrorIs records a failure when err does not match target under
// [errors.Is], which walks the chain of wrapped causes. A sentinel
// matches however deeply it was wrapped on the way up.
//
//	expect.ErrorIs(t, err, store.ErrNotFound, "Get reports a missing key")
//
// # Allocation contract
//
// A passing call on a sentinel wrapped twice allocates nothing.
func ErrorIs(tb assert.TB, err, target error, msg string) {
	tb.Helper()
	matcher.ErrorIs(tb, matcher.Soft, err, target, msg)
}

// ErrorIsNot records a failure when err matches target under
// [errors.Is].
//
// Use it where two sentinels must not match each other, because a caller
// cannot tell two cases apart when one sentinel matches the other.
//
// # Allocation contract
//
// A passing call on a sentinel wrapped twice allocates nothing.
func ErrorIsNot(tb assert.TB, err, target error, msg string) {
	tb.Helper()
	matcher.ErrorIsNot(tb, matcher.Soft, err, target, msg)
}

// ErrorAs finds the first error of type T in err's chain and returns
// it, and records a failure when the chain has none.
//
//	if notFound := expect.ErrorAs[*store.NotFoundError](t, err, "Get reports a missing key"); notFound != nil {
//	    expect.Equal(t, notFound.Key, "absent", "and names the key")
//	}
//
// On failure the test continues and ErrorAs returns the zero T. For a
// pointer type T that is nil, so check the result before reading
// through it.
//
// T is an interface type or a type that implements error, as [errors.As]
// requires of its target. Any other T ends the call with a fault that
// fails the test.
//
// # Allocation contract
//
// A passing call allocates only the target that [errors.As] fills.
func ErrorAs[T any](tb assert.TB, err error, msg string) T {
	tb.Helper()
	return matcher.ErrorAs[T](tb, matcher.Soft, err, msg)
}
