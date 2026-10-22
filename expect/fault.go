// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// FaultReporter is a [TB] that takes a fault as the error it is, the fault
// reporter of [go.dokimi.dev/assert]. [Recorder] satisfies it and keeps
// every fault.
type FaultReporter = assert.FaultReporter

// Fault ends the call of a helper with err, a fault: an argument or an
// environment that the helper refuses. It passes err to the first seat of
// tb's chain that satisfies [FaultReporter], with ending true. For a chain
// without one, it sends the writer's text of err through Fatalf of tb. A
// fault ends the call on this surface as well, because the call checked
// nothing and a test that ran on would meet err again. Fault does not write
// a call record, because the call is not a call of an assertion.
//
// Fault may not return, because Fatalf of a test stops it. A seat that takes
// the fault, such as a [Recorder], returns, so the helper returns after
// Fault:
//
//	if h.restart == nil {
//	    expect.Fault(h, fmt.Errorf("%w: %s calls Restart without conform.Restartable", ErrOption, h.check.ID))
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
		panic("expect: Fault(tb, nil) states no fault")
	}
	tb.Helper()
	matcher.End(tb, err)
}
