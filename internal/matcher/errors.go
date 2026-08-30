// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "errors"

// NoError reports when err is not nil, naming the error it got.
//
// Use it for an operation that must succeed. Where the failure itself
// is the subject, reach for [HasError] or [ErrorIs].
func NoError(seat Seat, mode Mode, err error, msg string) {
	seat.Helper()
	if err != nil {
		Report(seat, mode, "%s: unexpected error: %v", msg, err)
	}
}

// HasError reports when err is nil.
//
// It asks only that something failed. Where which failure matters, and
// it usually does, [ErrorIs] says so and this does not.
func HasError(seat Seat, mode Mode, err error, msg string) {
	seat.Helper()
	if err == nil {
		Report(seat, mode, "%s: expected an error, got nil", msg)
	}
}

// ErrorIs reports when err does not match target under [errors.Is],
// which walks the chain of wrapped causes. A sentinel therefore
// matches however deeply it was wrapped on the way up.
func ErrorIs(seat Seat, mode Mode, err, target error, msg string) {
	seat.Helper()
	if !errors.Is(err, target) {
		Report(seat, mode, "%s: got error %v, want one matching %v", msg, err, target)
	}
}

// ErrorIsNot reports when err matches target under [errors.Is]. Use it
// to hold two sentinels apart, where one matching the other would make
// a caller unable to tell the cases apart.
func ErrorIsNot(seat Seat, mode Mode, err, target error, msg string) {
	seat.Helper()
	if errors.Is(err, target) {
		Report(seat, mode, "%s: error %v matches %v, want them distinct", msg, err, target)
	}
}

// ErrorAs finds the first error of type T in err's chain and returns
// it, reporting when the chain holds none.
//
// The returned value is the zero T on failure, so a recording seat
// carries on with a usable value rather than a nil dereference. Under
// an aborting seat the call does not return at all.
//
//	notFound := matcher.ErrorAs[*store.NotFoundError](seat, matcher.Fatal, err,
//	    "Get reports a missing key")
//	matcher.Equal(seat, matcher.Fatal, notFound.Key, "absent", "and names the key")
func ErrorAs[T any](seat Seat, mode Mode, err error, msg string) T {
	seat.Helper()

	var target T
	if !errors.As(err, &target) {
		Report(seat, mode, "%s: got error %v, want one of type %T", msg, err, target)
	}
	return target
}
