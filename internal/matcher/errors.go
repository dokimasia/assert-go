// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"errors"
	"reflect"

	"go.dokimi.dev/assert/internal/fault"
)

// errorType is the type of the interface error.
var errorType = reflect.TypeFor[error]()

// The error assertions take got as an error or nil, as a function of a
// surface passes it. A chain passes its value, so got can be a value of
// another type, which fails each of them.

// NoError reports when got is not nil: an error, which it names, or a value
// that is no error.
//
// Use it for an operation that must succeed. Where the failure itself
// is the subject, use [HasError] or [ErrorIs].
//
// # Allocation contract
//
// A passing call allocates nothing.
func NoError(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()
	if got != nil {
		Fail(seat, mode, "err-absent", msg, map[string]any{gotField: got})
		return
	}
	Pass(seat, mode, "err-absent", msg)
}

// HasError reports when got is no error: nil, or a value of another type.
//
// It checks only that something failed. To check which failure it was,
// use [ErrorIs].
//
// # Allocation contract
//
// A passing call allocates nothing.
func HasError(seat Seat, mode Mode, got any, msg string) {
	seat.Helper()
	if _, isError := got.(error); !isError {
		Fail(seat, mode, "err-present", msg, nil)
		return
	}
	Pass(seat, mode, "err-present", msg)
}

// ErrorIs reports when got does not match target under [errors.Is], which
// walks the chain of wrapped causes, so a sentinel matches at any depth of
// wrapping. A value that is no error matches no target.
//
// # Allocation contract
//
// A passing call on a sentinel wrapped twice allocates nothing.
func ErrorIs(seat Seat, mode Mode, got any, target error, msg string) {
	seat.Helper()
	if err, isError := got.(error); got != nil && !isError || !errors.Is(err, target) {
		Fail(seat, mode, "err-is", msg, map[string]any{wantField: target, gotField: got})
		return
	}
	Pass(seat, mode, "err-is", msg)
}

// ErrorIsNot reports when got matches target under [errors.Is]. Use it
// for two sentinels that a caller must tell apart, where an error of one
// must not match the other. A value that is no error fails too.
//
// # Allocation contract
//
// A passing call on a sentinel wrapped twice allocates nothing.
func ErrorIsNot(seat Seat, mode Mode, got any, target error, msg string) {
	seat.Helper()
	if err, isError := got.(error); got != nil && !isError || errors.Is(err, target) {
		Fail(seat, mode, "err-is-not", msg, map[string]any{gotField: got})
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
// T is an interface type or a type that implements error, as [errors.As]
// requires of its target. Any other T ends the call with a fault, as
// [Fault] ends it, and ErrorAs then returns the zero T.
//
// # Allocation contract
//
// A passing call allocates only the target that [errors.As] fills.
func ErrorAs[T any](seat Seat, mode Mode, err error, msg string) T {
	seat.Helper()

	var target T
	if t := reflect.TypeFor[T](); t.Kind() != reflect.Interface && !t.Implements(errorType) {
		Fault(seat, mode, "err-as", msg, fault.New("the type %v is no interface and does not implement error", t))
		return target
	}
	if errors.As(err, &target) {
		Pass(seat, mode, "err-as", msg)
	} else {
		Fail(seat, mode, "err-as", msg, map[string]any{wantField: target, gotField: err})
	}
	return target
}
