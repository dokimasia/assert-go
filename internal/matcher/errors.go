// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "errors"

// NoError reports when err is not nil, naming the error it got.
//
// Use it for an operation that must succeed. Where the failure itself
// is the subject, use [HasError] or [ErrorIs].
//
// # Allocation contract
//
// A passing call allocates nothing.
func NoError(seat Seat, mode Mode, err error, msg string) {
	seat.Helper()
	if err != nil {
		Fail(seat, mode, "err-absent", msg, map[string]any{"got": err})
		return
	}
	Pass(seat, mode, "err-absent", msg)
}

// HasError reports when err is nil.
//
// It checks only that something failed. To check which failure it was,
// use [ErrorIs].
//
// # Allocation contract
//
// A passing call allocates nothing.
func HasError(seat Seat, mode Mode, err error, msg string) {
	seat.Helper()
	if err == nil {
		Fail(seat, mode, "err-present", msg, nil)
		return
	}
	Pass(seat, mode, "err-present", msg)
}

// ErrorIs reports when err does not match target under [errors.Is],
// which walks the chain of wrapped causes, so a sentinel matches at any
// depth of wrapping.
//
// # Allocation contract
//
// A passing call on a sentinel wrapped twice allocates nothing.
func ErrorIs(seat Seat, mode Mode, err, target error, msg string) {
	seat.Helper()
	if !errors.Is(err, target) {
		Fail(seat, mode, "err-is", msg, map[string]any{"want": target, "got": err})
		return
	}
	Pass(seat, mode, "err-is", msg)
}

// ErrorIsNot reports when err matches target under [errors.Is]. Use it
// for two sentinels that a caller must tell apart, where an error of one
// must not match the other.
//
// # Allocation contract
//
// A passing call on a sentinel wrapped twice allocates nothing.
func ErrorIsNot(seat Seat, mode Mode, err, target error, msg string) {
	seat.Helper()
	if errors.Is(err, target) {
		Fail(seat, mode, "err-is-not", msg, map[string]any{"got": err})
		return
	}
	Pass(seat, mode, "err-is-not", msg)
}

// ErrorAs returns the first error of type T in err's chain, and reports
// when the chain contains none.
//
// On a failure it returns the zero T, which is nil for a pointer type.
// Under [Fatal] it may not return.
//
//	notFound := matcher.ErrorAs[*store.NotFoundError](seat, matcher.Fatal, err,
//	    "Get reports a missing key")
//	matcher.Equal(seat, matcher.Fatal, notFound.Key, "absent", "and names the key")
//
// # Allocation contract
//
// A passing call allocates only the target that [errors.As] fills.
func ErrorAs[T any](seat Seat, mode Mode, err error, msg string) T {
	seat.Helper()

	var target T
	if !errors.As(err, &target) {
		Fail(seat, mode, "err-as", msg, map[string]any{"want": target, "got": err})
		return target
	}
	Pass(seat, mode, "err-as", msg)
	return target
}
