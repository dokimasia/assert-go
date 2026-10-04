// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
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
)

// Entry is one entry of a case of [Settings.Draws]: the label of a draw,
// and the value that the draw decodes to.
type Entry struct {
	// Label is the label of the draw.
	Label string
	// Value is a value of the draw's generator, or the value that the
	// typed literal of one decodes to.
	Value any
}

// inverting is the provider of a case of [Settings.Draws]. Each draw takes
// the next entry, and its requests take the choices that decode to the
// entry's value under the draw's generator. A request with no choice left,
// and every request once the entries run out, takes its target, as a replay
// that runs out of choices does.
type inverting struct {
	// entries are the entries, in draw order.
	entries []Entry
	// taken is the number of entries that draws have taken.
	taken int
	// queue are the choices of the current draw that no request has taken.
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
// The refusal of an entry of another label is a fault at the entry's label,
// and the refusal of a value is the fault of inverse at the entry's value.
// The path of a refusal starts at the entry's index. A draw past the last
// entry takes no choice.
func (d *inverting) next(label string, inverse func(any) ([]choice.Choice, error)) ([]choice.Choice, error) {
	if d.taken == len(d.entries) {
		return nil, nil
	}
	at := fault.Index(d.taken)
	entry := d.entries[d.taken]
	d.taken++
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
