// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"testing"
	"time"
)

// The waiting suites spend real time. These are the smallest values
// that still tell the outcomes apart, so a suite stays quick.
const (
	// ShortTimeout is long enough for a few attempts and short enough
	// that the timeout cases do not slow a run.
	ShortTimeout = 50 * time.Millisecond
	// ShortInterval is the gap between attempts.
	ShortInterval = 5 * time.Millisecond
	// PatientTimeout is for cases that must not time out; it bounds a
	// hang rather than pacing a retry.
	PatientTimeout = 5 * time.Second
	// SettlesAfter is how many attempts a converging subject takes.
	SettlesAfter = 3
)

// InnerReason is what a failing attempt reports, so a suite can
// require the outer failure to carry it.
const InnerReason = "matchertest: the attempt's own reason"

// EventuallyInvoke calls a surface's retrying assertion.
//
// The surface builds the body: it calls failing to learn whether this
// attempt should fail, and if so reports InnerReason through whatever
// seat it hands the body. That keeps the body's seat type out of this
// package, which is what lets one suite serve surfaces that name it
// differently.
type EventuallyInvoke func(seat *Seat, timeout, interval time.Duration, failing func() bool, msg string)

// PredicateInvoke calls a surface's retrying predicate assertion.
type PredicateInvoke func(seat *Seat, timeout time.Duration, pred func() bool, msg string)

// RunEventually drives invoke against every case a retrying assertion
// must produce.
func RunEventually(t *testing.T, invoke EventuallyInvoke) {
	t.Helper()

	t.Run("a body that passes first time reports nothing", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, ShortTimeout, ShortInterval, func() bool { return false }, contractMsg)
		checkOutcome(t, seat, false, nil)
	})

	t.Run("a body that passes on a later attempt reports nothing", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		seat := &Seat{}
		invoke(seat, PatientTimeout, ShortInterval, func() bool {
			attempts++
			return attempts < SettlesAfter
		}, contractMsg)

		checkOutcome(t, seat, false, nil)
		if attempts < SettlesAfter {
			t.Fatalf("ran %d attempts, want at least %d; it did not retry", attempts, SettlesAfter)
		}
	})

	t.Run("a body that never passes reports the attempt's own failure", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, ShortTimeout, ShortInterval, func() bool { return true }, contractMsg)
		checkOutcome(t, seat, true, []string{InnerReason})
	})

	t.Run("the body runs at least once however short the timeout", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		seat := &Seat{}
		invoke(seat, 0, ShortInterval, func() bool {
			attempts++
			return false
		}, contractMsg)

		checkOutcome(t, seat, false, nil)
		if attempts == 0 {
			t.Fatal("the body never ran")
		}
	})
}

// RunEventuallyTrue drives invoke against every case a retrying
// predicate assertion must produce.
func RunEventuallyTrue(t *testing.T, invoke PredicateInvoke) {
	t.Helper()

	t.Run("a predicate true first time reports nothing", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, ShortTimeout, func() bool { return true }, contractMsg)
		checkOutcome(t, seat, false, nil)
	})

	t.Run("a predicate that becomes true reports nothing", func(t *testing.T) {
		t.Parallel()

		calls := 0
		seat := &Seat{}
		invoke(seat, PatientTimeout, func() bool {
			calls++
			return calls >= SettlesAfter
		}, contractMsg)

		checkOutcome(t, seat, false, nil)
	})

	t.Run("a predicate never true reports the timeout", func(t *testing.T) {
		t.Parallel()

		seat := &Seat{}
		invoke(seat, ShortTimeout, func() bool { return false }, contractMsg)
		checkOutcome(t, seat, true, []string{"still false"})
	})
}
