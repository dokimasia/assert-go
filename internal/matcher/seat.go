// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// Seat is where a matcher sends a failure.
//
// A matcher never calls a test framework directly. It reports through
// a Seat, so one comparison serves a real test, a benchmark and a
// recorder without knowing which it has. The public TB interface
// declares the same three methods and satisfies this one structurally,
// so neither package imports the other for its seam.
type Seat interface {
	// Helper marks the calling frame as a helper, so a failure is
	// attributed to the caller's line rather than to the matcher.
	Helper()
	// Fatalf records a failure and stops the test. It may not return.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}

// Mode selects which of a [Seat]'s two failure methods a matcher uses.
// The zero value is [Fatal].
type Mode int

const (
	// Fatal reports through [Seat.Fatalf], stopping the test at the
	// first failure.
	Fatal Mode = iota
	// Soft reports through [Seat.Errorf], so the test runs on and
	// later failures are reported too.
	Soft
)

// Report marks the calling frame as a helper and sends one failure to
// seat, through [Seat.Fatalf] under [Fatal] and [Seat.Errorf] under
// [Soft].
//
// Report does not decide whether anything failed. A matcher calls it
// only once its own comparison has failed, so every call produces
// exactly one reported failure. Under [Fatal] it may not return.
func Report(seat Seat, mode Mode, format string, args ...any) {
	seat.Helper()
	if mode == Soft {
		seat.Errorf(format, args...)
		return
	}
	seat.Fatalf(format, args...)
}

// Fail sends one record to seat.
//
// A seat satisfying [Reporter] receives the record; any other seat
// receives the sentence [Render] makes of it. The call site is read
// here, so a matcher does not have to count frames.
//
// Fail does not decide whether anything failed. A matcher calls it
// only once its own comparison has failed. Under [Fatal] it may not
// return.
func Fail(seat Seat, mode Mode, assertion, contract string, detail map[string]any) {
	seat.Helper()
	f := Failure{
		Assertion: assertion,
		Contract:  contract,
		Detail:    detail,
		Where:     site(callerDepth),
	}
	if r, ok := seat.(Reporter); ok {
		r.Report(f, mode == Fatal)
		return
	}
	Report(seat, mode, "%s", Render(f))
}

// callerDepth is how many frames sit between a caller's line and the
// runtime.Caller inside site: the matcher that called Fail, Fail
// itself, and the assertion the caller wrote.
const callerDepth = 3
