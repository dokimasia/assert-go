// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import (
	"time"

	"go.dokimi.dev/assert"
)

// clock is the clock of a scheduler, as its tasks read it. It holds one
// pointer, so an interface stores it without an allocation.
type clock struct {
	// s is the scheduler of the clock.
	s *Scheduler
}

// Clock returns the clock of the scheduler. The clock starts at midnight UTC
// 2000-01-01, the instant at which a testing/synctest bubble starts, and only
// [Scheduler.Advance] moves it. Its Now returns an instant in UTC. Its Sleep
// parks the running task until the clock has passed the duration, and returns
// at once for a duration of zero or less. A sleeping task becomes ready after
// every ready task once the clock reads its deadline or a later instant, and
// keeps its priority under PCT.
//
// # Panics
//
// The Sleep of the clock panics outside a task for a positive duration,
// because only the body advances the clock.
//
// # Allocation contract
//
// Clock allocates nothing, and neither do the Now and the Sleep of the clock.
func (s *Scheduler) Clock() assert.Clock {
	return clock{s: s}
}

// Now returns the instant of the scheduler's clock.
func (c clock) Now() time.Time {
	return c.s.now
}

// Sleep parks the running task until the scheduler's clock has passed d,
// and returns at once for a d of zero or less.
func (c clock) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	s := c.s
	t := s.current
	if t == nil {
		panic("stateful: Sleep outside a task waits for a clock that only the body moves")
	}
	t.deadline = s.now.Add(d)
	s.park(t)
}
