// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// NoError stops the test when err is not nil, naming the error.
//
// Use it for an operation that must succeed. Where the failure is the
// subject, use [ErrorIs].
//
// # Allocation contract
//
// A passing call allocates nothing.
func NoError(tb TB, err error, msg string) {
	tb.Helper()
	matcher.NoError(tb, matcher.Fatal, err, msg)
}

// HasError stops the test when err is nil.
//
// It checks only that something failed. Where the kind of failure
// matters, use [ErrorIs], which checks the kind.
//
// # Allocation contract
//
// A passing call allocates nothing.
func HasError(tb TB, err error, msg string) {
	tb.Helper()
	matcher.HasError(tb, matcher.Fatal, err, msg)
}

// ErrorIs stops the test when err does not match target under
// [errors.Is], which walks the chain of wrapped causes. A sentinel
// matches however deeply it was wrapped on the way up.
//
//	assert.ErrorIs(t, err, store.ErrNotFound, "Get reports a missing key")
//
// # Allocation contract
//
// A passing call on a sentinel wrapped twice allocates nothing.
func ErrorIs(tb TB, err, target error, msg string) {
	tb.Helper()
	matcher.ErrorIs(tb, matcher.Fatal, err, target, msg)
}

// ErrorIsNot stops the test when err matches target under [errors.Is].
//
// Use it where two sentinels must not match each other, because a caller
// cannot tell two cases apart when one sentinel matches the other.
//
// # Allocation contract
//
// A passing call on a sentinel wrapped twice allocates nothing.
func ErrorIsNot(tb TB, err, target error, msg string) {
	tb.Helper()
	matcher.ErrorIsNot(tb, matcher.Fatal, err, target, msg)
}

// ErrorAs finds the first error of type T in err's chain and returns
// it, and stops the test when the chain has none.
//
//	notFound := assert.ErrorAs[*store.NotFoundError](t, err, "Get reports a missing key")
//	assert.Equal(t, notFound.Key, "absent", "and names the key")
//
// On failure the test stops, so a caller reads the returned value only
// after a match.
//
// T is an interface type or a type that implements error, as [errors.As]
// requires of its target. Any other T ends the call with a fault that
// fails the test.
//
// # Allocation contract
//
// A passing call allocates only the target that [errors.As] fills.
func ErrorAs[T any](tb TB, err error, msg string) T {
	tb.Helper()
	return matcher.ErrorAs[T](tb, matcher.Fatal, err, msg)
}
