// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "go.dokimi.dev/assert/internal/wrap"

// End ends a call that reports no verdict, such as a call of a helper that
// prepares a test, with err, a fault. It passes err to the first seat of the
// chain of seat that satisfies [FaultReporter]. For a chain without one, it
// sends the writer's text of err through Fatalf of seat. The chain is the
// seat and the seats that it wraps, as [wrap.As] walks it. End writes no
// call record, because the call is no call of an assertion. It may not
// return.
//
// # Allocation contract
//
// End allocates nothing besides what the ReportFault of seat allocates. For
// any other seat, it allocates the writer's text of err.
func End(seat Seat, err error) {
	seat.Helper()
	if reporter, ok := wrap.As[FaultReporter](seat); ok {
		reporter.ReportFault(err, true)
		return
	}
	seat.Fatalf("%s", writer.Fault(err))
}
