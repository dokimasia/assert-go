// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package realtime

import (
	"syscall"
	"time"
)

// bubbleGap is how far the clock of the time package can be behind the wall
// time of the system before Now takes it for the fake clock of a
// testing/synctest bubble. A bubble's clock starts at midnight UTC
// 2000-01-01.
const bubbleGap = 24 * time.Hour

// Clock reads the time that passes in the process.
type Clock struct{}

// Now returns the instant that time.Now returns, with its monotonic
// reading. Inside a testing/synctest bubble, whose fake clock is more than a
// day behind the wall time of the system, it returns the wall time of the
// system without a monotonic reading.
//
// # Allocation contract
//
// Now allocates nothing.
func (Clock) Now() time.Time {
	now := time.Now()
	var tv syscall.Timeval
	// Gettimeofday fails only for a pointer outside the address space.
	_ = syscall.Gettimeofday(&tv)
	wall := time.Unix(tv.Unix())
	if wall.Sub(now) > bubbleGap {
		return wall
	}
	return now
}
