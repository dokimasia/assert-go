// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// End ends a call that reports no verdict, such as a call of a helper that
// prepares a test, with err, a fault. It passes err to a seat that satisfies
// [FaultReporter], and sends the writer's text of err to any other seat
// through Fatalf. It writes no call record, because the call is no call of an
// assertion. It may not return.
//
// # Allocation contract
//
// End allocates nothing besides what the ReportFault of seat allocates. For
// any other seat, it allocates the writer's text of err.
func End(seat Seat, err error) {
	seat.Helper()
	if reporter, ok := seat.(FaultReporter); ok {
		reporter.ReportFault(err, true)
		return
	}
	seat.Fatalf("%s", writer.Fault(err))
}
