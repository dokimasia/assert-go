// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"fmt"
	"time"
)

// Outcome is what one client of [Concurrently] did.
type Outcome struct {
	// Client is the client's number, from 0.
	Client int
	// Finished reports whether the client's body ended before the time of
	// Concurrently passed.
	Finished bool
	// Output is what the body returned, when it finished.
	Output any
	// Error is the error that the body returned, when it finished.
	Error error
}

// exit is how one body ended: its outcome, and the value of its panic.
type exit struct {
	// outcome is the body's outcome.
	outcome Outcome
	// panicked reports whether the body panicked, with the value panic.
	panicked bool
	// panic is the value of the body's panic.
	panic any
}

// Concurrently starts clients copies of body, the i-th with client number
// i, from 0. It releases them together once it has started every copy, and
// waits until every body has ended or within has passed on the platform
// clock, counted from the release. It returns one outcome per client, in
// client order.
//
// A body that is still running when within passes has an outcome that is
// not Finished. Its goroutine runs on, because a goroutine cannot be
// stopped from outside, and a call that it opened in a history is pending
// there. A body that ends its goroutine with runtime.Goexit, as
// t.FailNow does, has finished with no output and no error.
//
// # Panics
//
// Concurrently panics for fewer than one client and for a negative time. It
// recovers a panic of a body, waits for the other bodies as it waits for
// any, and then panics on the caller's goroutine with the value of the
// panic of the lowest-numbered client that panicked. A body that panics
// after within passed ends without a trace.
//
// # Allocation contract
//
// Concurrently allocates a goroutine and its arguments for each client, the
// release channel, the channel of exits, the timer, and the outcomes that it
// returns: 8 allocations for one client.
func Concurrently(clients int, within time.Duration, body func(client int) (any, error)) []Outcome {
	if clients < 1 {
		panic(fmt.Sprintf("history: Concurrently(%d, %v) starts no client", clients, within))
	}
	if within < 0 {
		panic(fmt.Sprintf("history: Concurrently(%d, %v) waits a negative time", clients, within))
	}
	release := make(chan struct{})
	ended := make(chan exit, clients)
	for client := range clients {
		go runClient(client, body, release, ended)
	}
	close(release)
	timer := time.NewTimer(within)
	defer timer.Stop()
	outcomes := make([]Outcome, clients)
	for i := range outcomes {
		outcomes[i].Client = i
	}
	raised := -1
	var value any
collect:
	for range clients {
		select {
		case e := <-ended:
			outcomes[e.outcome.Client] = e.outcome
			if e.panicked && (raised < 0 || e.outcome.Client < raised) {
				raised, value = e.outcome.Client, e.panic
			}
		case <-timer.C:
			break collect
		}
	}
	if raised >= 0 {
		panic(value)
	}
	return outcomes
}

// runClient runs body for client once release closes, and sends how it
// ended to ended, which has room for every client's exit.
func runClient(client int, body func(int) (any, error), release <-chan struct{}, ended chan<- exit) {
	e := exit{outcome: Outcome{Client: client, Finished: true}}
	defer func() {
		if v := recover(); v != nil {
			e.panicked, e.panic = true, v
		}
		ended <- e
	}()
	<-release
	e.outcome.Output, e.outcome.Error = body(client)
}
