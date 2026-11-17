// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/wrap"
)

// failer is a seat that reports whether its test, its case or its body
// failed, as testing.TB has Failed.
type failer interface {
	// Failed reports whether the test, the case or the body failed.
	Failed() bool
}

// Failed returns the result of the method Failed of the first seat of the
// chain of seat that has the method, as [wrap.As] walks the chain. A chain
// without the method ends the call of op with a fault through [End], and
// Failed returns false where End returns.
//
// # Allocation contract
//
// Failed allocates nothing besides what the method Failed allocates. A chain
// without the method allocates the fault and its text.
func Failed(seat Seat, op string) bool {
	seat.Helper()
	if f, ok := wrap.As[failer](seat); ok {
		return f.Failed()
	}
	End(seat, fault.In(op, fault.New("the seat has no Failed")))
	return false
}
