// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// True reports when cond is false.
//
// The failure carries msg alone. A boolean that came out wrong has no
// detail worth printing, which is why the message is required: it is
// the only thing telling a reader what was supposed to hold.
func True(seat Seat, mode Mode, cond bool, msg string) {
	seat.Helper()
	if !cond {
		Report(seat, mode, "%s: expected true, got false", msg)
	}
}

// False reports when cond is true. See [True] for why the failure
// carries no detail beyond msg.
func False(seat Seat, mode Mode, cond bool, msg string) {
	seat.Helper()
	if cond {
		Report(seat, mode, "%s: expected false, got true", msg)
	}
}
