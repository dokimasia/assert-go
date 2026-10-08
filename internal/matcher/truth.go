// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// True reports when cond is false.
//
// The failure states msg alone, because a boolean has no detail. The
// message is required: it is the only text that tells a reader what the
// test expected.
//
// # Allocation contract
//
// A passing call allocates nothing.
func True(seat Seat, mode Mode, cond bool, msg string) {
	seat.Helper()
	if !cond {
		Fail(seat, mode, "true", msg, nil)
		return
	}
	Pass(seat, mode, "true", msg)
}

// False reports when cond is true. As for [True], the failure states msg
// alone.
//
// # Allocation contract
//
// A passing call allocates nothing.
func False(seat Seat, mode Mode, cond bool, msg string) {
	seat.Helper()
	if cond {
		Fail(seat, mode, "false", msg, nil)
		return
	}
	Pass(seat, mode, "false", msg)
}
