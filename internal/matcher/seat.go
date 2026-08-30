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
