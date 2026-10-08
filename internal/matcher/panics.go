// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// Panics runs fn and reports when it returns without panicking. It
// returns whatever fn panicked with, so a caller can assert on the
// reason as well as the fact.
//
//	reason := matcher.Panics(seat, matcher.Fatal, func() { MustGet("") },
//	    "MustGet rejects an empty key")
//	matcher.Contains(seat, matcher.Fatal, reason, "empty key",
//	    "and states which argument was wrong")
//
// A panic with the value nil passes. Since Go 1.21 the runtime replaces
// the nil with a [runtime.PanicNilError], and Panics returns that error.
//
// Read the seat, not the return, to learn whether fn panicked. Under
// GODEBUG=panicnil=1 a panic with nil returns nil, as a call of fn that
// did not panic does.
//
// # Allocation contract
//
// A passing call allocates nothing besides what fn and its panic allocate.
func Panics(seat Seat, mode Mode, fn func(), msg string) (recovered any) {
	seat.Helper()

	panicked := true
	func() {
		defer func() { recovered = recover() }()
		fn()
		panicked = false
	}()

	if !panicked {
		Fail(seat, mode, "throws", msg, nil)
		return nil
	}
	Pass(seat, mode, "throws", msg)
	return recovered
}

// NotPanics runs fn and reports when it panics, naming what it
// panicked with.
//
// The panic is recovered, so a subject that panics fails its test and the
// rest of the run goes on.
//
// This is the assertion for a function that may fail: it may return an
// error, and it must not panic. Pass a call with a zero value or an absent
// argument to state that the subject handles it.
//
// A fn that ends its goroutine through runtime.Goexit, as a fatal failure
// on a recorder does, neither panics nor returns, and the call states no
// verdict.
//
// # Allocation contract
//
// A passing call allocates nothing besides what fn allocates.
func NotPanics(seat Seat, mode Mode, fn func(), msg string) {
	seat.Helper()

	returned := false
	defer func() {
		if r := recover(); r != nil {
			Fail(seat, mode, "not-throws", msg, map[string]any{gotField: r})
			return
		}
		if returned {
			Pass(seat, mode, "not-throws", msg)
		}
	}()
	fn()
	returned = true
}
