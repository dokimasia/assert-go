// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// Panics runs fn and reports when it returns without panicking. It
// returns whatever fn panicked with, so a caller can assert on the
// reason as well as the fact.
//
//	reason := matcher.Panics(seat, matcher.Fatal, func() { MustGet("") },
//	    "MustGet rejects an empty key")
//	matcher.Contains(seat, matcher.Fatal, reason, "empty key",
//	    "and says which argument was wrong")
//
// A panic carrying nil is still a panic and passes. Since Go 1.21 the
// runtime replaces the nil with a [runtime.PanicNilError], so the
// returned value is that error rather than nil.
//
// Read the seat, not the return, to learn whether fn panicked. A
// function that panics with a nil-valued variable of some other type
// yields exactly what a function that did not panic would.
func Panics(seat Seat, mode Mode, fn func(), msg string) (recovered any) {
	seat.Helper()

	panicked := true
	func() {
		defer func() { recovered = recover() }()
		fn()
		panicked = false
	}()

	if !panicked {
		Report(seat, mode, "%s: expected a panic, the function returned", msg)
	}
	return recovered
}

// NotPanics runs fn and reports when it panics, naming what it
// panicked with.
//
// The panic is recovered rather than allowed to unwind, so one
// misbehaving subject fails its test instead of taking the run down.
//
// This is the assertion for a function that may legitimately fail:
// returning an error is fine, crashing is not. Pass a call with a zero
// value or an absent argument to state that the subject handles it.
func NotPanics(seat Seat, mode Mode, fn func(), msg string) {
	seat.Helper()

	defer func() {
		if r := recover(); r != nil {
			Report(seat, mode, "%s: unexpected panic: %v", msg, r)
		}
	}()
	fn()
}
