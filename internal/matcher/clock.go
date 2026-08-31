// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"sync"
	"time"
)

// Clock is where an assertion reads time.
//
// An assertion that waits, retries or measures reads it here rather
// than calling the runtime, so a test can supply time it controls and
// a busy machine cannot make the assertion flaky.
type Clock interface {
	// Now answers the current instant.
	Now() time.Time
	// Sleep blocks until the duration has passed on this clock.
	Sleep(d time.Duration)
}

// Clocked is a [Seat] that carries a clock.
//
// [testing.TB] declares three methods and can never grow a fourth, so
// a clock reaches an assertion through a second interface a seat may
// also satisfy. A seat that does not satisfy it gets [System].
type Clocked interface {
	Clock() Clock
}

// System reads the runtime clock. It is what an assertion gets when
// the seat supplies nothing.
type System struct{}

// Now answers the runtime's current instant.
func (System) Now() time.Time { return time.Now() }

// Sleep blocks for d against the runtime clock.
func (System) Sleep(d time.Duration) { time.Sleep(d) }

// ClockOf answers the clock seat carries, or [System] when it carries
// none.
func ClockOf(seat Seat) Clock {
	if c, ok := seat.(Clocked); ok {
		if held := c.Clock(); held != nil {
			return held
		}
	}
	return System{}
}

// Controlled is a clock that moves only when a test advances it.
//
// Now answers what Advance last left it at, and Sleep blocks until the
// clock has passed the duration rather than until the wall has. An
// assertion that retries advances this clock between attempts rather
// than sleeping against it, so a body settling on the third attempt
// costs three attempts and no waiting.
//
// Every method is safe to call from any goroutine.
type Controlled struct {
	mu      sync.Mutex
	woke    *sync.Cond
	instant time.Time
}

// NewControlled returns a clock reading start until it is advanced.
func NewControlled(start time.Time) *Controlled {
	c := &Controlled{instant: start}
	c.woke = sync.NewCond(&c.mu)
	return c
}

// Now answers the instant this clock was last advanced to.
func (c *Controlled) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.instant
}

// Advance moves the clock forward by d and wakes everything sleeping
// that the new instant has passed. A negative d does not move it
// backwards; time on this clock only goes forward.
func (c *Controlled) Advance(d time.Duration) {
	if d <= 0 {
		return
	}
	c.mu.Lock()
	c.instant = c.instant.Add(d)
	c.mu.Unlock()
	c.woke.Broadcast()
}

// Sleep blocks until the clock has passed d.
//
// It returns at once when d is not positive. Otherwise it waits for
// another goroutine to advance the clock, so a test that sleeps on the
// only goroutine it has blocks until something advances it.
//
// The duration is measured from the instant Sleep reads, so a caller
// racing Sleep against Advance on two goroutines cannot say which
// instant it slept from. Assertions do not hit this: one that retries
// advances the clock itself, on the goroutine it is already running
// on.
func (c *Controlled) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	until := c.instant.Add(d)
	for c.instant.Before(until) {
		c.woke.Wait()
	}
}

// wait moves time forward by d.
//
// A clock a test controls is advanced, because nothing else will move
// it while this call is running. Any other clock is slept against.
func wait(c Clock, d time.Duration) {
	if a, ok := c.(interface{ Advance(time.Duration) }); ok {
		a.Advance(d)
		return
	}
	c.Sleep(d)
}
