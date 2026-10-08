// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

// Part is a part of a machine's steps. prop's Part converts from it, value
// for value.
type Part uint8

const (
	// SwarmPart is the swarm's choice of the actions that a case keeps.
	SwarmPart Part = 0
	// SetupPart is the check and the invariant before the first step.
	SetupPart Part = 1
	// SequentialPart is the sequential steps on client 0.
	SequentialPart Part = 2
	// ConcurrentPart is the concurrent section: the steps that it lists, and
	// their run.
	ConcurrentPart Part = 3
	// DrainPart is the steps of the drain.
	DrainPart Part = 4
	// SettlePart is the settle check and the last invariant.
	SettlePart Part = 5
)

// Place is the part of a machine's steps that was running, and its step.
type Place struct {
	// Part is the part that was running.
	Part Part
	// Positioned reports whether Position is set.
	Positioned bool
	// Acting reports whether Action is set.
	Acting bool
	// Position is the step's position in its part, from 0, and in the swarm
	// the position of the action whose keep choice it is, when Positioned.
	// No step runs in setup, in settle, and while a concurrent section runs
	// the steps that it listed.
	Position int
	// Action is the action of the swarm choice or of the step, when Acting.
	// A step that has not chosen its action has none.
	Action string
}

// Where is where a case made a request or observed a fingerprint: the draw
// and the part of a machine's steps that were running.
type Where struct {
	// Label is the label of the innermost draw that was running, when
	// Drawing.
	Label string
	// Drawing reports whether a draw was running.
	Drawing bool
	// Placed reports whether a part of a machine was running.
	Placed bool
	// Place is the part and the step of a machine that were running, when
	// Placed.
	Place Place
}

// SetPlace makes p the place of every request that the case makes and every
// fingerprint that it observes, until the next SetPlace or ClearPlace. A
// machine's steps set the place of each part and each step as they run. The
// divergence of a flaky run states the place where the run differed.
//
// # Allocation contract
//
// SetPlace allocates nothing.
func (c *Case) SetPlace(p Place) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.place, c.placed = p, true
}

// ClearPlace leaves the case outside a machine's steps.
//
// # Allocation contract
//
// ClearPlace allocates nothing.
func (c *Case) ClearPlace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.place, c.placed = Place{}, false
}

// where returns where the case is: the label of its innermost draw and its
// place. The caller has locked c.mu.
func (c *Case) where() Where {
	return Where{Label: c.label, Drawing: c.drawing, Place: c.place, Placed: c.placed}
}

// drawState is the innermost draw of a case: its label, when drawing.
type drawState struct {
	// label is the draw's label.
	label string
	// drawing reports whether a draw runs.
	drawing bool
}

// openDraw makes label the label of the case's innermost draw, and returns
// the innermost draw before it, which the closeDraw that the draw defers
// restores.
func (c *Case) openDraw(label string) drawState {
	c.mu.Lock()
	defer c.mu.Unlock()
	outer := drawState{label: c.label, drawing: c.drawing}
	c.label, c.drawing = label, true
	return outer
}

// closeDraw ends the innermost draw, and makes outer the innermost draw
// again.
func (c *Case) closeDraw(outer drawState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.label, c.drawing = outer.label, outer.drawing
}

// requestWhere returns where the case made the request at index, and the
// zero Where past the last request. A case that does not keep its wheres
// has none.
func (c *Case) requestWhere(index int) Where {
	c.mu.Lock()
	defer c.mu.Unlock()
	if index >= len(c.wheres) {
		return Where{}
	}
	return c.wheres[index]
}

// fingerprintWhere returns where the case observed the fingerprint at
// index, and the zero Where past the last fingerprint. A case that does not
// keep its wheres has none.
func (c *Case) fingerprintWhere(index int) Where {
	c.mu.Lock()
	defer c.mu.Unlock()
	if index >= len(c.observed) {
		return Where{}
	}
	return c.observed[index]
}
