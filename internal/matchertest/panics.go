// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import "testing"

// PanicReason is what the panicking fixtures panic with, so a case can
// require the failure or the returned value to carry it.
const PanicReason = "matchertest: the stated reason"

// PanicsInvoke calls a surface's panics assertion, returning whatever
// the assertion yields.
type PanicsInvoke func(seat *Seat, fn func(), msg string) any

// NotPanicsInvoke calls a surface's not-panics assertion.
type NotPanicsInvoke func(seat *Seat, fn func(), msg string)

// RunPanics drives invoke against every case a panics assertion must
// produce, including that it yields what the function panicked with.
func RunPanics(t *testing.T, invoke PanicsInvoke) {
	t.Helper()

	t.Run("a panicking function passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, func() { panic(PanicReason) }, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("it yields the panic value", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		got := invoke(seat, func() { panic(PanicReason) }, contractMsg)

		if got != PanicReason {
			t.Fatalf("yielded %v, want %q", got, PanicReason)
		}
	})

	t.Run("a panic carrying a nil-valued error still counts", func(t *testing.T) {
		t.Parallel()

		// A nil of some concrete type, rather than a literal nil: the
		// runtime rewrites the latter, so it would not exercise the
		// case it appears to.
		var nothing *TypedError

		seat := &Seat{}
		invoke(seat, func() { panic(nothing) }, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a returning function reports", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, func() {}, contractMsg)
		checkOutcome(t, seat, Case{Fails: true, Assertion: "throws"})
	})
}

// RunNotPanics drives invoke against every case a not-panics assertion
// must produce, including that a panic does not escape it.
func RunNotPanics(t *testing.T, invoke NotPanicsInvoke) {
	t.Helper()

	t.Run("a returning function passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, func() {}, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a function that fails without crashing passes", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, func() { _ = ErrSample }, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("a panicking function reports what it panicked with", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, func() { panic(PanicReason) }, contractMsg)
		checkOutcome(t, seat, Case{
			Fails: true, Assertion: "not-throws",
			Detail: map[string]any{"got": PanicReason},
		})
	})

	t.Run("the panic does not escape", func(t *testing.T) {
		t.Parallel()

		// Reaching the assertion below is the case: an unrecovered
		// panic would have taken the test binary down instead.
		seat := &Seat{}
		invoke(seat, func() { panic(PanicReason) }, contractMsg)

		if !seat.Failed() {
			t.Fatal("reported nothing for a panicking function")
		}
	})
}
