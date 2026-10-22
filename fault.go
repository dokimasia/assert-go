// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// FaultReporter is a [TB] that takes a fault as the error it is, rather than
// as the writer's text of it, as a [Reporter] takes a failure's record.
//
// A fault is an error that ends a call without a verdict: an argument or an
// environment that the call refuses. The first FaultReporter of a seat's
// chain receives a fault that ends a call with ending true, and a fault that
// a call notes and runs on past, such as a damaged file of a property's
// store, with ending false. A chain without a FaultReporter receives the
// writer's text of an ending fault through Fatalf, and the text of a noted
// fault in the log of the first seat that has one. [Recorder] satisfies it
// and keeps every fault.
type FaultReporter = matcher.FaultReporter

// Fault ends the call of a helper with err, a fault: an argument or an
// environment that the helper refuses. It passes err to the first seat of
// tb's chain that satisfies [FaultReporter], with ending true. For a chain
// without one, it sends the writer's text of err through Fatalf of tb. The
// faults of this module end their calls the same way. Fault does not write
// a call record, because the call is no call of an assertion.
//
// Fault may not return, because Fatalf of a test stops it. A seat that takes
// the fault, such as a [Recorder], returns, so the helper returns after
// Fault:
//
//	if h.restart == nil {
//	    assert.Fault(h, fmt.Errorf("%w: %s calls Restart without conform.Restartable", ErrOption, h.check.ID))
//	    return
//	}
//
// It panics for a nil err, because a nil error is not a fault.
//
// # Allocation contract
//
// Fault allocates nothing besides what the ReportFault of tb allocates. For
// any other seat, it allocates the writer's text of err.
func Fault(tb TB, err error) {
	if err == nil {
		panic("assert: Fault(tb, nil) states no fault")
	}
	tb.Helper()
	matcher.End(tb, err)
}
