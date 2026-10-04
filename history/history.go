// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"fmt"
	"slices"
	"sync"

	"go.dokimi.dev/assert/internal/literal"
)

// History is the events of one history, in one recording order that every
// client shares.
//
// Each client starts on a process of its own, and processes are numbered
// in the order of their first invocation. A call that completes as
// [Unknown] may still be in progress inside the subject, so its client
// continues on a new process. Every process has at most one open call.
//
// The history keeps the values it receives and does not copy them. A caller
// records a copy of a value that the subject or the test changes later.
//
// The zero History is an empty history, as [New] returns one.
//
// # Concurrency
//
// Every method of a History and of each [Call] that it returns is safe for
// concurrent use. One mutex orders every invocation and completion, so the
// events are in one order that every client observes. Call a precedes call
// b only when a's completion took the mutex before b's invocation did, so a
// returned before b started.
type History struct {
	// mu orders the events.
	mu sync.Mutex
	// events are the recorded events, in recording order.
	events []Event
	// ids are the identities of each event's keys, the texts of their typed
	// literals, and nil for a completion.
	ids [][]string
	// open maps each client with an open call to the index of its
	// invocation.
	open map[int]int
	// process maps each client to its process.
	process map[int]int
	// processes is the number of processes so far.
	processes int
}

// New returns an empty history.
//
// # Allocation contract
//
// New allocates the history: one allocation.
func New() *History {
	return &History{}
}

// Invoke records an invocation of operation with args by client, and returns
// the call through which the client records the completion. keys lists the
// keys that the call touches, and a call without keys touches every key.
//
// Two keys are one key when their typed literals are equal, so the int 1 and
// the float 1.0 are two keys, and the ints 1 and int64(1) are one.
//
// # Allocation contract
//
// Invoke allocates the identities of the keys, the typed literal of each
// key, and the growth of the history's events. The first invocation of a
// history allocates its maps of open calls and processes as well.
//
// # Panics
//
// Invoke panics when client has a call open, and when a key has no typed
// literal.
func (h *History) Invoke(client int, operation string, args []any, keys ...any) Call {
	ids := make([]string, len(keys))
	for i, key := range keys {
		raw, ok := literal.Encode(key)
		if !ok {
			panic(fmt.Sprintf("history: Invoke(%d, %q) states the key %T, which no typed literal states",
				client, operation, key))
		}
		ids[i] = string(raw)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if open, busy := h.open[client]; busy {
		panic(fmt.Sprintf("history: Invoke(%d, %q) while call %d of client %d is open",
			client, operation, open, client))
	}
	if h.open == nil {
		h.open, h.process = map[int]int{}, map[int]int{}
	}
	process, started := h.process[client]
	if !started {
		process = h.processes
		h.processes++
		h.process[client] = process
	}
	index := len(h.events)
	h.events = append(h.events, Event{
		Index: index, Kind: Invoke, Call: index, Client: client, Process: process,
		Operation: operation, Args: args, Keys: keys,
	})
	h.ids = append(h.ids, ids)
	h.open[client] = index
	return Call{history: h, index: index}
}

// Events returns the recorded events in recording order, without a gap: the
// events with the indices 0 to n - 1 for every event recorded before the
// call.
//
// # Allocation contract
//
// Events allocates the copy of the events that it returns.
func (h *History) Events() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.events)
}

// recorded returns the events and the identities of their keys, each a copy
// taken under the mutex.
func (h *History) recorded() ([]Event, [][]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.events), slices.Clone(h.ids)
}

// complete records the completion of kind of the call whose invocation is at
// index call, with output or err.
//
// It panics when the call has completed already.
func (h *History) complete(call int, kind Kind, output any, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	invocation := h.events[call]
	if open, busy := h.open[invocation.Client]; !busy || open != call {
		panic(fmt.Sprintf("history: call %d completes a second time", call))
	}
	delete(h.open, invocation.Client)
	if kind == Unknown {
		delete(h.process, invocation.Client)
	}
	index := len(h.events)
	h.events = append(h.events, Event{
		Index: index, Kind: kind, Call: call, Client: invocation.Client, Process: invocation.Process,
		Output: output, Error: err,
	})
	h.ids = append(h.ids, nil)
}
