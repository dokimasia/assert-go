// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record

// Slot is a call that runs a body: it took its number before the calls of
// its body, and writes its record when it ends. A nil Slot is the slot of
// a call that is not recorded, and each of its methods does nothing.
//
// # Concurrency
//
// The call that started a slot takes the runs of its body and writes its
// record, on one goroutine.
type Slot struct {
	// entry is the slot's place in the Calls that keeps its record.
	entry *entry
	// runs counts the runs of the body taken so far.
	runs int
}

// Begin starts the record of a call on seat that runs a body, and returns
// its slot: the call takes its number now, before the calls of its body.
// It returns nil when [Of] returns nil for seat.
//
// # Allocation contract
//
// Begin allocates the slot and its entry for a recorded call, and nothing
// otherwise.
func Begin(seat any) *Slot {
	calls := Of(seat)
	if calls == nil {
		return nil
	}
	e := &entry{}
	calls.add(e)
	return &Slot{entry: e}
}

// Take takes the result of the next run of the slot's body, whose calls
// body keeps, and moves them to the Calls that keeps the slot: each call
// of the run, with the slot's call as its parent, the run's number and
// phase. A call of a body inside the run keeps its own parent. body is
// empty afterwards.
//
// Take numbers every run that it takes, a run without calls included.
//
// # Allocation contract
//
// Take allocates what numbering the moved records allocates.
func (s *Slot) Take(body *Calls, phase Phase) {
	if s == nil {
		return
	}
	body.mu.Lock()
	entries := body.entries
	body.entries = nil
	body.mu.Unlock()
	s.runs++
	home := s.entry.home.Load()
	for _, e := range entries {
		if e.parent == nil {
			e.parent, e.run, e.phase = s.entry, s.runs, phase
		}
		home.add(e)
	}
}

// Write ends the slot's call with its record c, which the Calls that keeps
// the slot writes under the number that the call took when it began.
//
// # Panics
//
// It panics when the detail of c is no JSON.
//
// # Allocation contract
//
// Write allocates the encoded record in a Calls of a test or of a
// recorder.
func (s *Slot) Write(c Call) {
	if s == nil {
		return
	}
	home := s.entry.home.Load()
	home.mu.Lock()
	defer home.mu.Unlock()
	s.entry.call, s.entry.done = c, true
	if s.entry.seq != 0 {
		home.emit(s.entry)
	}
}
