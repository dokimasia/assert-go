// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// logger is a seat with a log, as testing.T, testing.B and testing.F have
// one.
type logger interface {
	// Logf formats its arguments as fmt.Sprintf does and adds the text to
	// the test's log.
	Logf(format string, args ...any)
}

// Note writes text into the log of seat, when seat has a log: a note that
// the body of a property attached to its failing case, which the property
// writes before it reports its failure. A seat without a log receives
// nothing.
//
// # Allocation contract
//
// Note allocates nothing besides what the Logf of seat allocates.
func Note(seat Seat, text string) {
	seat.Helper()
	if l, ok := seat.(logger); ok {
		l.Logf("%s", text)
	}
}

// NoteFault notes err on seat: a fault that does not end the call, such as
// a damaged file of a store, which the run skips. It passes err to a seat
// that satisfies [FaultReporter], and writes the writer's text of err into
// the log of any other seat that has a log. A seat without either receives
// nothing.
//
// # Allocation contract
//
// NoteFault allocates nothing besides what the ReportFault of seat
// allocates. For a seat with a log, it allocates the writer's text of err.
func NoteFault(seat Seat, err error) {
	seat.Helper()
	if reporter, ok := seat.(FaultReporter); ok {
		reporter.ReportFault(err, false)
		return
	}
	Note(seat, writer.Fault(err))
}
