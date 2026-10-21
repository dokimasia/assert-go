// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "go.dokimi.dev/assert/internal/wrap"

// logger is a seat with a log, as testing.T, testing.B and testing.F have
// one.
type logger interface {
	// Logf formats its arguments as fmt.Sprintf does and adds the text to
	// the test's log.
	Logf(format string, args ...any)
}

// Note writes text into the log of the first seat of the chain of seat that
// has a log: a note that the body of a property attached to its failing
// case, which the property writes before it reports its failure. The chain
// is the seat and the seats that it wraps, as [wrap.As] walks it. A chain
// without a log receives nothing.
//
// # Allocation contract
//
// Note allocates nothing besides what the Logf of seat allocates.
func Note(seat Seat, text string) {
	seat.Helper()
	if l, ok := wrap.As[logger](seat); ok {
		l.Logf("%s", text)
	}
}

// NoteFault notes err on seat: a fault that does not end the call, such as
// a damaged file of a store, which the run skips. It passes err to the first
// seat of the chain of seat that satisfies [FaultReporter]. A chain without
// one receives the writer's text of err in its log, as [Note] writes it, and
// a chain without either receives nothing.
//
// # Allocation contract
//
// NoteFault allocates nothing besides what the ReportFault of seat
// allocates. For a seat with a log, it allocates the writer's text of err.
func NoteFault(seat Seat, err error) {
	seat.Helper()
	if reporter, ok := wrap.As[FaultReporter](seat); ok {
		reporter.ReportFault(err, false)
		return
	}
	Note(seat, writer.Fault(err))
}
