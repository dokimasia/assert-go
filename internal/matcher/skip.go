// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/wrap"
)

// Skip calls the method Skipf with format and args on the first seat of the
// chain of seat that has the method. [wrap.As] walks the chain. On a chain
// without the method, Skip ends the call of op with a fault through [End].
// Under a seat of the package testing, Skip does not return.
//
// # Allocation contract
//
// Skip allocates nothing besides what the method Skipf allocates. On a chain
// without the method, it allocates the fault and its text.
func Skip(seat Seat, op, format string, args ...any) {
	seat.Helper()
	if s, ok := wrap.As[skipper](seat); ok {
		s.Skipf(format, args...)
		return
	}
	End(seat, fault.In(op, fault.New("the seat has no Skipf")))
}
