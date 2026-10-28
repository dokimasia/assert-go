// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package record

// Observation is one assertion call that a seat observes: the id of the
// assertion and the contract that the caller passed.
type Observation struct {
	// Assertion is the id of the assertion, as the definition names it.
	Assertion string
	// Contract is the caller's message, unchanged.
	Contract string
}

// Observer is a seat that observes each assertion call, a passing one
// included. The first Observer of a seat's chain receives every call that a
// recorded run records, whatever [Variable] states, once and before the seat
// receives the failure, the fault or the skip of the call. It receives the
// calls of a body when the call that ran the body takes them, so the calls of
// a property's cases arrive in the order of a run on one worker. ObserveCall
// may run on any goroutine of the test.
type Observer interface {
	// ObserveCall receives one assertion call.
	ObserveCall(call Observation)
}

// observed returns the Calls of a call on a chain whose observer is o. It
// passes each call to o, and hands its record to to, the Calls that keeps or
// writes the record, or to nobody for a nil to.
func observed(o Observer, to *Calls) *Calls {
	return &Calls{state: observing, observer: o, to: to}
}

// observe passes the call of e to its observer, once the call has ended. It
// passes each call once, and nothing for an entry without an observer.
func observe(e *entry) {
	if e.observer == nil || !e.done || e.observed {
		return
	}
	e.observed = true
	e.observer.ObserveCall(Observation{Assertion: e.call.Assertion, Contract: e.call.Contract})
}
