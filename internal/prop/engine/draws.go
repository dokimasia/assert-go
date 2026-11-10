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

// Entry is one entry of a trace, such as the entries of [Settings.Draws]
// and the trace of a [Stored] case. A draw entry has the label of a draw and
// the value that the draw decodes to. A step entry has a step of a machine.
type Entry struct {
	// Label is the label of the draw.
	Label string
	// Value is a value of the draw's generator, or the value that the
	// typed literal of one decodes to.
	Value any
	// Unstated reports a draw entry without a value, as a stored entry
	// records a value that no typed literal states. A draw refuses such an
	// entry, and a composite takes it by its label alone.
	Unstated bool
	// Step is the step of a step entry, which states no label and no value,
	// and nil for a draw entry.
	Step *MachineStep
}

// inverting is the provider of a case that follows a trace. Each draw takes
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
// A composite passes a nil inverse once its draws have taken their entries,
// and takes its entry by its label alone. The refusal of a step entry and of
// an entry of another label is a fault at the entry's label. The refusal of
// an entry without a value is a fault at the entry's value, and so is the
// fault of inverse for a value. The path of a refusal starts at the entry's
// index. A draw past the last entry takes no choice.
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
	if inverse == nil {
		return nil, nil
	}
	if entry.Unstated {
		return nil, fault.At(fault.New("the entry records no value"), at, fault.Field(valueMember))
	}
	choices, err := inverse(entry.Value)
	if err != nil {
		return nil, fault.At(err, at, fault.Field(valueMember))
	}
	return choices, nil
}

// enterDraw starts a draw labelled label in a case that follows a trace,
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

// leaveComposite takes the entry of a composite labelled label in a case
// that follows a trace, once the draws of the composite have taken theirs.
// A refused entry ends the case and the calling goroutine.
func (c *Case) leaveComposite(label string) {
	_, refusal := c.inverting.next(label, nil)
	if refusal == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refusal = refusal
	c.halt(refused)
}

// executeTrace calls body once on a case that follows entries, outside the
// case tree. It returns how the case ended, with the number of entries that
// the draws and the steps of the case took.
func executeTrace(body Body, s Settings, entries []Entry) (Execution, int) {
	p := &inverting{entries: entries}
	c := newCase(p, s, nil)
	c.inverting = p
	e := finish(c, body)
	return e, p.taken
}

// Trace is the trace that a case follows, such as the case of
// [Settings.Draws]. The draws and the steps of the case take its entries in
// order. A machine turns each step entry into the choices of one step. It
// reads the next entry with [Trace.Next], and takes it with [Trace.Take] or
// refuses it with [Trace.Refuse]. Take serves the choices of the step to
// the requests of the step.
//
// # Concurrency
//
// Every method is safe for concurrent use. A machine calls them on the
// goroutine of the body.
type Trace struct {
	// c is the case that follows the trace.
	c *Case
}

// Trace returns the trace that the case follows, and false for a case that
// follows none. The case of [Settings.Draws] follows a trace, and so does
// the case of the trace of a [Stored] case. No other case follows one.
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
