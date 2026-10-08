// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

// Checkpoint keeps the search of the last check of a whole history that
// passed with it, so that the next check of the history continues that
// search instead of searching from the first call. [Resume] passes it to a
// check. The zero Checkpoint keeps nothing.
//
// A machine's check after each step keeps its search in a Checkpoint, so
// each check searches the calls since the check before it. Over a case of n
// sequential calls, the spec's Next runs once for each call.
//
// # Concurrency
//
// A Checkpoint is safe for one check at a time.
type Checkpoint[S any] struct {
	// history is the history that the kept search checked.
	history *History
	// events is the number of events that the kept search read.
	events int
	// search is the kept search, and nil when the checkpoint keeps none.
	search *search[S]
}

// grown returns the search that cp keeps, grown by the calls that h has
// recorded since, through ops under limits and the deadline d. It returns
// nil when the search cannot continue as a search of the whole history
// does: cp keeps none, or keeps one of another history or of other limits,
// or h has completed a call that the kept search read as pending, or the
// memo contains more configurations than the grown search may store.
func (cp *Checkpoint[S]) grown(h *History, ops operations[S], limits config, d *deadline) *search[S] {
	s := cp.search
	if s == nil || cp.history != h {
		return nil
	}
	events, ids := h.recordedFrom(cp.events)
	more, kept := callsOf(events, ids, cp.events)
	if !kept || !s.fits(len(more), limits) {
		return nil
	}
	s.grow(ops, more, limits, d)
	cp.events += len(events)
	return s
}

// resumed returns the detail of a check of the whole of h through ops under
// c and the deadline d that continues the search that cp keeps, when cp can,
// and that searches every call otherwise. It keeps the search in cp when the
// check passes, and keeps none when it does not.
func resumed[S any](h *History, ops operations[S], c config, d *deadline, cp *Checkpoint[S]) (detail[S], error) {
	s := cp.grown(h, ops, c, d)
	if s == nil {
		events, ids := h.recordedFrom(0)
		calls, _ := callsOf(events, ids, 0)
		s = newSearch(ops, partition{keys: []any{}, calls: calls}, c, d)
		cp.history, cp.events = h, len(events)
	}
	cp.search = nil
	e := s.runApart()
	if e.err != nil {
		return detail[S]{}, e.err
	}
	if e.verdict != Passed {
		return reported(partition{keys: []any{}, calls: s.calls}, e, 1, e.steps), nil
	}
	cp.search = s
	return detail[S]{outcome: Passed, partitions: min(len(s.calls), 1), steps: e.steps, final: e.states}, nil
}
