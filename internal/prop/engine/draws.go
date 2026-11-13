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
// its entry, and its requests take the choices that decode to the entry's
// value under the draw's generator. A machine turns each step entry into the
// choices of one step, which [Trace] serves to its requests. A request with
// no choice left, and every request once the entries run out, takes its
// target, as a replay that runs out of choices does.
type inverting struct {
	// entries are the entries, in request order.
	entries []Entry
	// taken is the number of entries that draws and steps have taken.
	taken int
	// open are the draws in progress, the innermost last. The first stands
	// for the requests outside every draw, such as the steps of a machine.
	open []openDraw
}

// openDraw is a draw in progress in a case that follows a trace.
type openDraw struct {
	// queue are the choices of the draw that no request has taken.
	queue []choice.Choice
	// end is the index of the entry that a draw through a composite takes
	// once it decodes, and -1 for every other draw.
	end int
	// byLabel reports a composite, which takes the next entry by its label
	// alone once it decodes.
	byLabel bool
}

// value returns the next choice of the innermost open draw, fitted to r's
// bounds, or r's target when none is left.
func (d *inverting) value(r request, _ int) choice.Choice {
	open := &d.open[len(d.open)-1]
	if len(open.queue) == 0 {
		return r.bounds.Target()
	}
	v := open.queue[0]
	open.queue = open.queue[1:]
	return r.bounds.Coerce(v)
}

// enter opens a draw labelled label with the choices that inverse returns
// for the value of its entry, or returns the refusal of the entry. A draw
// through a composite takes the first entry with its label up to the next
// step entry. When inverse reports a value through a composite, the draw
// takes that entry once it decodes. Every other draw takes the next entry
// now. The refusal of a step entry and of an entry of another label is a
// fault at the label of the next entry. The refusal of an entry without a
// value is a fault at the entry's value, and so is the fault of inverse for
// a value. The path of a refusal starts at the entry's index. A draw past
// the last entry takes no choice.
func (d *inverting) enter(label string, through bool, inverse func(any) ([]choice.Choice, bool, error)) error {
	if d.taken == len(d.entries) {
		d.open = append(d.open, openDraw{end: -1})
		return nil
	}
	at := d.taken
	if through {
		at = d.find(label)
	}
	entry := d.entries[at]
	if entry.Step != nil || entry.Label != label {
		return d.refusal(at, label)
	}
	if entry.Unstated {
		return fault.At(fault.New("the entry records no value"), fault.Index(at), fault.Field(valueMember))
	}
	choices, composite, err := inverse(entry.Value)
	if err != nil {
		return fault.At(err, fault.Index(at), fault.Field(valueMember))
	}
	if composite {
		d.open = append(d.open, openDraw{queue: choices, end: at})
		return nil
	}
	if at != d.taken {
		return d.refusal(d.taken, label)
	}
	d.taken++
	d.open = append(d.open, openDraw{queue: choices, end: -1})
	return nil
}

// leave ends the innermost open draw, labelled label, and takes its entry
// once it decodes. A composite takes the next entry by its label alone. A
// draw through a composite takes its entry, which must be the next one, once
// the draws inside it have taken the entries before it. When a draw inside
// took that entry, the refusal is at the entry's label.
func (d *inverting) leave(label string) error {
	ended := d.open[len(d.open)-1]
	d.open = d.open[:len(d.open)-1]
	switch {
	case ended.byLabel && d.taken < len(d.entries):
		return d.take(label)
	case d.taken < ended.end:
		return d.refusal(d.taken, label)
	case ended.end >= 0 && d.taken > ended.end:
		refusal := fault.New("a draw inside the draw labelled %q takes the entry of the draw", label)
		return fault.At(refusal, fault.Index(ended.end), fault.Field(labelMember))
	case ended.end >= 0:
		d.taken++
	}
	return nil
}

// take takes the next entry, a draw entry labelled label, or returns its
// refusal.
func (d *inverting) take(label string) error {
	entry := d.entries[d.taken]
	if entry.Step != nil || entry.Label != label {
		return d.refusal(d.taken, label)
	}
	d.taken++
	return nil
}

// find returns the index of the first draw entry labelled label from the
// next entry up to the next step entry, and the index of the next entry when
// there is none.
func (d *inverting) find(label string) int {
	for at := d.taken; at < len(d.entries); at++ {
		entry := d.entries[at]
		if entry.Step != nil {
			break
		}
		if entry.Label == label {
			return at
		}
	}
	return d.taken
}

// refusal returns the refusal of the entry with index for a draw labelled
// label: a fault at the entry's label that names the step of a step entry or
// the label of a draw entry.
func (d *inverting) refusal(index int, label string) error {
	entry := d.entries[index]
	refusal := fault.New("the draw labelled %q takes the entry labelled %q", label, entry.Label)
	if entry.Step != nil {
		refusal = fault.New("the draw labelled %q takes the step entry of %q", label, entry.Step.Action)
	}
	return fault.At(refusal, fault.Index(index), fault.Field(labelMember))
}

// enterDraw starts a draw labelled label in a case that follows a trace,
// whose generator's choices for a value inverse returns. through reports a
// draw through a composite. A refused entry ends the case and the calling
// goroutine.
func (c *Case) enterDraw(label string, through bool, inverse func(any) ([]choice.Choice, bool, error)) {
	refusal := c.inverting.enter(label, through, inverse)
	c.mu.Lock()
	defer c.mu.Unlock()
	if refusal != nil {
		c.refusal = refusal
		c.halt(refused)
	}
}

// enterComposite starts the draw of a composite in a case that follows a
// trace. The composite takes no entry until it decodes.
func (c *Case) enterComposite() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inverting.open = append(c.inverting.open, openDraw{end: -1, byLabel: true})
}

// leaveDraw ends the draw labelled label in a case that follows a trace,
// and takes its entry where it takes one once it decodes. A refused entry
// ends the case and the calling goroutine.
func (c *Case) leaveDraw(label string) {
	refusal := c.inverting.leave(label)
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
	p := &inverting{entries: entries, open: []openDraw{{end: -1}}}
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
	open := &t.c.inverting.open[len(t.c.inverting.open)-1]
	for _, v := range values {
		open.queue = append(open.queue, choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)})
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
