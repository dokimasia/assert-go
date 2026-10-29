// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"sync"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/wrap"
)

// cleaner is a seat that runs functions when its test, its case or its body
// ends, as testing.TB has them.
type cleaner interface {
	// Cleanup registers f to run when the test, the case or the body ends.
	Cleanup(f func())
}

// Cleanup registers f with the method Cleanup of the first seat of the chain
// of seat that has it, as [wrap.As] walks the chain. A chain without the
// method ends the call of op with a fault through [End], and f does not run.
//
// # Allocation contract
//
// Cleanup allocates nothing besides what the method Cleanup allocates. A
// chain without the method allocates the fault and its text.
func Cleanup(seat Seat, op string, f func()) {
	seat.Helper()
	if c, ok := wrap.As[cleaner](seat); ok {
		c.Cleanup(f)
		return
	}
	End(seat, fault.In(op, fault.New("the seat has no Cleanup")))
}

// Cleanups are the functions that a seat runs when its test or its body
// ends. The zero value keeps none. Every method is safe for concurrent use.
type Cleanups struct {
	mu sync.Mutex
	// fns are the functions that wait for a run, in the order of their
	// registration.
	fns []func()
	// closed records that [Cleanups.Close] ran the functions, so that a
	// function that arrives later runs at once.
	closed bool
}

// Add registers f for the next run. After [Cleanups.Close], Add runs f at
// once, on the calling goroutine.
//
// # Allocation contract
//
// Add allocates nothing besides the growth of the list.
func (c *Cleanups) Add(f func()) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		f()
		return
	}
	c.fns = append(c.fns, f)
	c.mu.Unlock()
}

// Run runs the registered functions, the last registered first, and forgets
// them. A function that a function registers runs in the same pass. Run calls
// each function in a frame of its own, so a function that ends the goroutine
// or panics leaves the earlier functions to run. A function that arrives
// after Run returns waits for the next run.
//
// # Allocation contract
//
// Run allocates nothing besides what the functions allocate.
func (c *Cleanups) Run() {
	c.run(false)
}

// Close runs the registered functions as [Cleanups.Run] does. Every function
// that arrives after them runs at once.
//
// # Allocation contract
//
// Close allocates nothing besides what the functions allocate.
func (c *Cleanups) Close() {
	c.run(true)
}

// run runs the last registered function, and then the rest through a
// deferred call of itself. Once no function is left, it marks the list
// closed when closing is true.
func (c *Cleanups) run(closing bool) {
	c.mu.Lock()
	last := len(c.fns) - 1
	if last < 0 {
		c.closed = c.closed || closing
		c.mu.Unlock()
		return
	}
	f := c.fns[last]
	c.fns[last] = nil
	c.fns = c.fns[:last]
	c.mu.Unlock()

	defer c.run(closing)
	f()
}
