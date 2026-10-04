// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"cmp"
	"errors"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
)

// fromIntervalsOp is the operation of FromIntervals, which names its faults.
const fromIntervalsOp = "history.FromIntervals"

// ErrInterval is the kind of the fault of an entry that [FromIntervals]
// refuses.
var ErrInterval = errors.New("history: an entry that from-intervals refuses")

// Interval is a call recorded with a start and an end on one clock, as a
// log of a system's calls states it.
type Interval struct {
	// Client is the client that made the call.
	Client int
	// Operation is the call's operation.
	Operation string
	// Args are the call's arguments.
	Args []any
	// Keys are the keys that the call touches. A call without keys touches
	// every key.
	Keys []any
	// Start is the time the call started.
	Start int64
	// End is the time the call completed. A pending call has no end.
	End int64
	// Kind is the call's completion kind: OK, Fail or Unknown, or Invoke for
	// a pending call, whose End is not read.
	Kind Kind
	// Output is what the call returned, for OK.
	Output any
	// Error is the call's error, for Fail and Unknown.
	Error error
}

// The phases of a moment, in the order of two moments at one time.
const (
	// invocation is the moment an entry starts.
	invocation = 0
	// completion is the moment an entry ends.
	completion = 1
)

// moment is the invocation or the completion of an entry, at its time.
type moment struct {
	// time is the time of the invocation or the completion.
	time int64
	// phase is invocation or completion.
	phase int
	// entry is the index of the entry.
	entry int
}

// FromIntervals returns the history of calls recorded with a start and an
// end on one clock. The events are in time order. At one time, an
// invocation comes before a completion, and the invocations, or the
// completions, keep the order of their entries. Each interval is closed,
// so two entries that share an instant overlap, and a pending entry
// overlaps every later entry of its client. Times are compared and never
// subtracted.
//
// # Errors
//
// It returns a fault of the kind [ErrInterval], with the operation
// history.FromIntervals and the index of the entry as its path, for the
// first entry in the given order that states no kind, ends before it
// starts, states a key that no typed literal states, or overlaps an earlier
// entry of its client.
//
// # Allocation contract
//
// FromIntervals allocates the moments of the entries, the index of each
// client's entries, and what a [History] allocates for the events.
func FromIntervals(entries []Interval) (*History, error) {
	earlier := map[int][]int{}
	for i, e := range entries {
		if err := e.refusal(entries, earlier[e.Client]); err != nil {
			return nil, fault.In(fromIntervalsOp, fault.At(err, fault.Index(i)))
		}
		earlier[e.Client] = append(earlier[e.Client], i)
	}
	moments := make([]moment, 0, 2*len(entries))
	for i, e := range entries {
		moments = append(moments, moment{time: e.Start, phase: invocation, entry: i})
		if e.Kind != Invoke {
			moments = append(moments, moment{time: e.End, phase: completion, entry: i})
		}
	}
	slices.SortFunc(moments, func(a, b moment) int {
		return cmp.Or(cmp.Compare(a.time, b.time), cmp.Compare(a.phase, b.phase), cmp.Compare(a.entry, b.entry))
	})
	h := New()
	calls := make([]Call, len(entries))
	for _, m := range moments {
		e := entries[m.entry]
		if m.phase == invocation {
			calls[m.entry] = h.Invoke(e.Client, e.Operation, e.Args, e.Keys...)
			continue
		}
		switch e.Kind {
		case OK:
			calls[m.entry].OK(e.Output)
		case Fail:
			calls[m.entry].Fail(e.Error)
		default:
			calls[m.entry].Unknown(e.Error)
		}
	}
	return h, nil
}

// refusal returns the fault of e, an entry of entries, whose client's
// earlier entries are at the indices earlier, and nil for an entry that
// FromIntervals accepts.
func (e Interval) refusal(entries []Interval, earlier []int) error {
	if !e.Kind.Valid() {
		return fault.Of(ErrInterval, "states no kind of event")
	}
	if e.Kind != Invoke && e.End < e.Start {
		return fault.Of(ErrInterval, "ends at %d, before it starts at %d", e.End, e.Start)
	}
	for _, key := range e.Keys {
		if _, ok := literal.Encode(key); !ok {
			return fault.Of(ErrInterval, "states the key %T, which no typed literal states", key)
		}
	}
	for _, j := range earlier {
		if overlap(entries[j], e) {
			return fault.Of(ErrInterval, "overlaps entry %d of client %d", j, e.Client)
		}
	}
	return nil
}

// overlap reports whether two closed intervals share an instant. A pending
// interval has no end.
func overlap(a, b Interval) bool {
	return (a.Kind == Invoke || b.Start <= a.End) && (b.Kind == Invoke || a.Start <= b.End)
}
