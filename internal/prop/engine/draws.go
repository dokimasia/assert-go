// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// The members of an entry of [Settings.Draws] as the definition names them,
// at which the path of a refusal ends.
const (
	// labelMember is the label of an entry.
	labelMember = "label"
	// valueMember is the value of an entry.
	valueMember = "value"
	// stepMember is the action of a step entry.
	stepMember = "step"
)

// Entry is one entry of a case of [Settings.Draws]: a draw entry, which
// states the label of a draw and the value that the draw decodes to, or a
// step entry, which states a step of a machine.
type Entry struct {
	// Label is the label of the draw.
	Label string
	// Value is a value of the draw's generator, or the value that the
	// typed literal of one decodes to.
	Value any
	// Step is the step of a step entry, which states no label and no value,
	// and nil for a draw entry.
	Step *MachineStep
}

// inverting is the provider of a case of [Settings.Draws]. Each draw takes
// the next entry, and its requests take the choices that decode to the
// entry's value under the draw's generator. A machine turns each step entry
// into the choices of one step, which [Trace] serves to its requests. A
// request with no choice left, and every request once the entries run out,
// takes its target, as a replay that runs out of choices does.
type inverting struct {
	// entries are the entries, in request order.
	entries []Entry
	// taken is the number of entries that draws and steps have taken.
	taken int
	// queue are the choices of the current draw or step that no request has
	// taken.
	queue []choice.Choice
}

// value returns the next choice of the current draw, fitted to r's bounds,
// or r's target when none is left.
func (d *inverting) value(r request, _ int) choice.Choice {
	if len(d.queue) == 0 {
		return r.bounds.Target()
	}
	v := d.queue[0]
	d.queue = d.queue[1:]
	return r.bounds.Coerce(v)
}

// next takes the entry of a draw labelled label, and returns the choices
// that inverse returns for the entry's value, or the refusal of the entry.
// The refusal of a step entry and of an entry of another label is a fault at
// the entry's label, and the refusal of a value is the fault of inverse at
// the entry's value. The path of a refusal starts at the entry's index. A
// draw past the last entry takes no choice.
func (d *inverting) next(label string, inverse func(any) ([]choice.Choice, error)) ([]choice.Choice, error) {
	if d.taken == len(d.entries) {
		return nil, nil
	}
	at := fault.Index(d.taken)
	entry := d.entries[d.taken]
	d.taken++
	if entry.Step != nil {
		refusal := fault.New("the draw labelled %q takes the step entry of %q", label, entry.Step.Action)
		return nil, fault.At(refusal, at, fault.Field(labelMember))
	}
	if entry.Label != label {
		refusal := fault.New("the draw labelled %q takes the entry labelled %q", label, entry.Label)
		return nil, fault.At(refusal, at, fault.Field(labelMember))
	}
	choices, err := inverse(entry.Value)
	if err != nil {
		return nil, fault.At(err, at, fault.Field(valueMember))
	}
	return choices, nil
}

// enterDraw starts a draw labelled label in the case of [Settings.Draws],
// whose generator's choices for a value inverse returns. The draw takes its
// entry's choices, and a refused entry ends the case and the calling
// goroutine.
func (c *Case) enterDraw(label string, inverse func(any) ([]choice.Choice, error)) {
	choices, refusal := c.inverting.next(label, inverse)
	c.mu.Lock()
	defer c.mu.Unlock()
	if refusal != nil {
		c.refusal = refusal
		c.halt(refused)
	}
	c.inverting.queue = choices
}

// executeDraws calls body once on the case of s.Draws, outside the case
// tree, and returns how the case ended.
func executeDraws(body Body, s Settings) Execution {
	p := &inverting{entries: s.Draws}
	c := newCase(p, s, nil)
	c.inverting = p
	return finish(c, body)
}

// Trace is the trace that the case of [Settings.Draws] follows: its entries,
// which its draws and its machine's steps take in order. A machine turns
// each step entry into the choices of one step. It reads the next entry with
// [Trace.Next], and takes it with [Trace.Take], which serves the choices of
// the step to the step's requests, or refuses it with [Trace.Refuse].
//
// # Concurrency
//
// Every method is safe for concurrent use. A machine calls them on the
// goroutine of the body.
type Trace struct {
	// c is the case of Settings.Draws.
	c *Case
}

// Trace returns the trace that the case follows, and false for a case that
// follows none, which every case but the case of [Settings.Draws] is.
func (c *Case) Trace() (Trace, bool) {
	return Trace{c: c}, c.inverting != nil
}

// Names reports whether a step entry from the next entry on names action.
func (t Trace) Names(action string) bool {
	t.c.mu.Lock()
	defer t.c.mu.Unlock()
	d := t.c.inverting
	names := func(e Entry) bool { return e.Step != nil && e.Step.Action == action }
	return slices.ContainsFunc(d.entries[d.taken:], names)
}

// Prepare serves the integer values to the next requests of the case, in
// order, before any target.
func (t Trace) Prepare(values ...uint64) {
	t.c.mu.Lock()
	defer t.c.mu.Unlock()
	d := t.c.inverting
	for _, v := range values {
		d.queue = append(d.queue, choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)})
	}
}

// Next returns the next entry when it is a step entry, and false when the
// next entry is a draw entry or the entries ran out.
func (t Trace) Next() (MachineStep, bool) {
	t.c.mu.Lock()
	defer t.c.mu.Unlock()
	d := t.c.inverting
	if d.taken == len(d.entries) || d.entries[d.taken].Step == nil {
		return MachineStep{}, false
	}
	return *d.entries[d.taken].Step, true
}

// Take takes the next entry, a step entry, and serves the integer values to
// the next requests of the case, in order, as [Trace.Prepare] does.
func (t Trace) Take(values ...uint64) {
	t.Prepare(values...)
	t.c.mu.Lock()
	defer t.c.mu.Unlock()
	t.c.inverting.taken++
}

// Refuse ends the case with the refusal of the next entry, a step entry
// that the machine cannot take at its position, and ends the calling
// goroutine. The refusal is a fault at the entry's step, with reason as its
// reason, and its path starts at the entry's index.
func (t Trace) Refuse(reason string) {
	c := t.c
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refusal = fault.At(fault.New("%s", reason), fault.Index(c.inverting.taken), fault.Field(stepMember))
	c.halt(refused)
}
