// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"sync"
	"time"
)

// Clock is where an assertion reads time.
//
// An assertion that waits, retries or measures reads the time here and
// not from the runtime, so a test can supply time that it controls, and
// a busy machine cannot make the assertion flaky.
type Clock interface {
	// Now returns the current instant.
	Now() time.Time
	// Sleep blocks until the duration has passed on this clock.
	Sleep(d time.Duration)
}

// Clocked is a [Seat] that supplies a clock.
//
// [testing.TB] declares three methods and can never declare a fourth, so
// a seat supplies a clock through this second interface. An assertion
// reads [System] when its seat does not satisfy Clocked.
type Clocked interface {
	Clock() Clock
}

// System reads the runtime clock. An assertion reads it when the seat
// supplies no clock.
type System struct{}

// Now returns the runtime's current instant.
func (System) Now() time.Time { return time.Now() }

// Sleep blocks for d against the runtime clock.
func (System) Sleep(d time.Duration) { time.Sleep(d) }

// ClockOf returns the clock that seat supplies, or [System] when it
// supplies none.
func ClockOf(seat Seat) Clock {
	if c, ok := seat.(Clocked); ok {
		if supplied := c.Clock(); supplied != nil {
			return supplied
		}
	}
	return System{}
}

// Controlled is a clock that moves only when a test advances it.
//
// Now returns the instant that Advance last moved it to, and Sleep blocks
// until the clock has passed the duration, whatever the wall clock reads.
// An assertion that retries advances this clock between attempts instead
// of sleeping against it, so a body that settles on the third attempt
// costs three attempts and no waiting.
//
// Every method is safe to call from any goroutine.
type Controlled struct {
	mu      sync.Mutex
	woke    *sync.Cond
	instant time.Time
}

// NewControlled returns a clock that reads start until it is advanced.
func NewControlled(start time.Time) *Controlled {
	c := &Controlled{instant: start}
	c.woke = sync.NewCond(&c.mu)
	return c
}

// Now returns the instant that this clock was last advanced to.
func (c *Controlled) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.instant
}

// Advance moves the clock forward by d, or by nothing when d is not
// positive, and wakes every sleeper.
func (c *Controlled) Advance(d time.Duration) {
	c.mu.Lock()
	c.instant = c.instant.Add(max(d, 0))
	c.mu.Unlock()
	c.woke.Broadcast()
}

// Sleep blocks until the clock has passed d.
//
// It returns at once when d is not positive. Otherwise it waits for
// another goroutine to advance the clock, so a test that sleeps on the
// only goroutine it has blocks until something advances the clock.
//
// The duration is measured from the instant that Sleep reads, so a
// caller that races Sleep against Advance on two goroutines cannot tell
// which instant Sleep measured from. An assertion that retries advances
// the clock itself, on its own goroutine, and does not race.
func (c *Controlled) Sleep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	until := c.instant.Add(d)
	for c.instant.Before(until) {
		c.woke.Wait()
	}
}

// wait moves time forward by d.
//
// A clock that a test controls is advanced, because nothing else moves
// it while this call runs. Any other clock is slept against.
func wait(c Clock, d time.Duration) {
	if a, ok := c.(interface{ Advance(time.Duration) }); ok {
		a.Advance(d)
		return
	}
	c.Sleep(d)
}
