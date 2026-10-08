// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package record

import (
	"sync"
	"sync/atomic"
)

// state is what Calls does with a record.
type state uint8

const (
	// off records nothing. It is the zero state.
	off state = iota
	// keeping numbers each record and keeps it, for a recorder.
	keeping
	// writing numbers each record and writes it through a test's Attr.
	writing
	// running keeps the records of one run of a body, unnumbered, until the
	// call that ran the body takes them.
	running
)

// entry is one call in Calls: its record, and its place in the calls of
// its test.
type entry struct {
	// call is the record. It is complete once done is set.
	call Call
	// done reports whether the call has ended. A call that runs a body
	// takes its number before it ends.
	done bool
	// parent is the entry of the call whose body this call ran in, and nil
	// for a call that ran in no body.
	parent *entry
	// run is the run of the parent's body that this call ran in.
	run int
	// phase is the phase of that run, for a call in a property's case.
	phase Phase
	// steps is how many steps the walk of the run's case had made when the
	// call reported its verdict.
	steps int
	// seq is the call's number, and 0 until a Calls of a test or of a
	// recorder numbers it.
	seq int
	// home is the Calls that keeps the entry now.
	home atomic.Pointer[Calls]
}

// Calls keeps the records of the calls of one test, one recorder or one run
// of a body.
//
// A seat of this module embeds a Calls through an unexported alias, and
// [Of] finds it there. A Calls has no exported method, so the seat that
// embeds it gains no exported name. The zero Calls records nothing:
// [Keep] makes it a recorder's, and [Run] the Calls of one run of a body.
//
// # Concurrency
//
// Every function of this package that takes a Calls is safe for concurrent
// use.
type Calls struct {
	// mu guards the fields below.
	mu sync.Mutex
	// state is what the Calls does with a record.
	state state
	// next is the number of the last record numbered, in a Calls that
	// keeps or writes.
	next int
	// lines are the encoded records that a recorder's Calls keeps, by their
	// number less one. A call that is still running a body has an empty
	// line.
	lines []string
	// sink is the test that a writing Calls writes through.
	sink attrSeat
	// entries are the calls of a run, in the order of their verdicts.
	entries []*entry
	// position returns how many steps the walk of the run's case has made,
	// and nil for a run whose calls are never cut.
	position func() int
}

// keeper is a seat of this module, which embeds a Calls.
type keeper interface {
	calls() *Calls
}

// calls returns c, which makes a seat that embeds a Calls a keeper.
func (c *Calls) calls() *Calls {
	return c
}

// Of returns the Calls that keeps the record of a call on seat, and nil
// when the call is not recorded:
//
//   - A seat of this module that embeds a Calls has its Calls, unless that
//     Calls records nothing.
//   - A test's seat, which declares Attr, Name and Output as testing.T,
//     testing.B and testing.F do, has the Calls of its test while [On]
//     reports true. A type that embeds the test's seat has the same Calls.
//     The package keeps the Calls of a test for the life of the process,
//     because a cleanup of the test can make calls until the test ends.
//   - Any other seat has none.
//
// # Allocation contract
//
// Of allocates nothing, except the Calls of a test on the test's first
// recorded call.
func Of(seat any) *Calls {
	if k, ok := seat.(keeper); ok {
		c := k.calls()
		c.mu.Lock()
		live := c.state != off
		c.mu.Unlock()
		if live {
			return c
		}
		return nil
	}
	return testCalls(seat)
}

// Add gives c its number in calls and writes it, or keeps it until the
// call that ran the body of calls takes it.
//
// # Panics
//
// It panics when the detail of c is no JSON.
//
// # Allocation contract
//
// Add allocates the entry of c, and in a Calls of a test or of a recorder
// its encoded record.
func Add(calls *Calls, c Call) {
	calls.add(&entry{call: c, done: true})
}

// add puts e in c: numbered and written when c keeps or writes, and kept
// unnumbered when c is the Calls of a run.
func (c *Calls) add(e *entry) {
	steps := 0
	if position := c.positionOf(); position != nil {
		steps = position()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e.home.Store(c)
	switch c.state {
	case running:
		e.steps = steps
		c.entries = append(c.entries, e)
	case keeping, writing:
		c.next++
		e.seq = c.next
		if e.done {
			c.emit(e)
		}
	case off:
		// A Calls that is off records nothing.
	}
}

// positionOf returns the function that states the steps of the run's walk,
// and nil for a Calls without one.
func (c *Calls) positionOf() func() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.position
}

// emit writes the record of e, a numbered entry whose call has ended, with
// c.mu locked.
func (c *Calls) emit(e *entry) {
	text := encode(e)
	if c.state == writing {
		write(c.sink, e.seq, text)
		return
	}
	for len(c.lines) < e.seq {
		c.lines = append(c.lines, "")
	}
	c.lines[e.seq-1] = text
}

// Keep makes c the Calls of a recorder, which keeps the record of every
// call it receives, numbered from 1. It drops what c kept before.
//
// # Allocation contract
//
// Keep allocates nothing.
func Keep(c *Calls) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state, c.next, c.lines, c.sink, c.entries, c.position = keeping, 0, nil, nil, nil, nil
}

// Lines returns the records that a recorder's c keeps, one line of JSON
// each, in the order of their numbers. A call that is still running a body
// has no line yet. The slice is a fresh copy.
//
// # Allocation contract
//
// Lines allocates the slice it returns.
func Lines(c *Calls) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.lines))
	for _, l := range c.lines {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// Run makes body the Calls of one run of the body of the call that slot
// started, with no record. A nil slot, the slot of a call whose record no
// Calls keeps, makes body record nothing. position states how many steps
// the walk of the run's case has made, for [Cut], and is nil for a run
// whose calls are never cut.
//
// # Allocation contract
//
// Run allocates nothing.
func Run(body *Calls, slot *Slot, position func() int) {
	body.mu.Lock()
	defer body.mu.Unlock()
	body.state, body.next, body.lines, body.sink, body.entries, body.position = off, 0, nil, nil, nil, nil
	if slot != nil {
		body.state, body.position = running, position
	}
}

// Cut drops the records of the calls that the run of body made at its
// walk's step steps and after it: the calls that a run on one worker does
// not make, because the case stops at that step there.
//
// # Allocation contract
//
// Cut allocates nothing.
func Cut(body *Calls, steps int) {
	body.mu.Lock()
	defer body.mu.Unlock()
	kept := body.entries[:0]
	for _, e := range body.entries {
		if e.steps < steps {
			kept = append(kept, e)
		}
	}
	clear(body.entries[len(kept):])
	body.entries = kept
}
