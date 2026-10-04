// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// Seat is where a matcher sends a failure or a fault.
//
// A matcher calls no test framework directly. It reports through a Seat,
// so one comparison works on a test, a benchmark and a recorder alike. The
// public TB interface declares the same three methods and satisfies this
// one structurally, so neither package imports the other.
type Seat interface {
	// Helper marks the calling frame as a helper, so a failure is
	// attributed to the caller's line rather than to the matcher.
	Helper()
	// Fatalf records a failure and stops the test. It may not return.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}

// FaultReporter is a [Seat] that takes a fault as the error it is, rather
// than as the writer's text of it, as a [Reporter] takes a failure's record.
//
// A FaultReporter receives the fault of a call that ends without a verdict,
// such as one of [Fault], with ending true, and a fault that [NoteFault]
// notes for a call that runs on, with ending false. Any other seat receives
// the writer's text of the first through Fatalf, and of the second through
// its log.
type FaultReporter interface {
	ReportFault(err error, ending bool)
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
