// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
)

// The allocation ceilings of the functions of a history.
const (
	// newAllocs is the ceiling of the allocations of New: the history.
	newAllocs = 2
	// invokeAllocs is the ceiling of the allocations of a first invocation
	// of one string key, with the history it invokes on.
	invokeAllocs = 20
	// eventsAllocs is the ceiling of the allocations of Events: the copy of
	// the events.
	eventsAllocs = 2
)

// The size of the reading of a history while clients record.
const (
	// recordingClients is the number of clients that record at once.
	recordingClients = 8
	// recordedCalls is the number of calls that each client records.
	recordedCalls = 200
)

// TestHistory checks the recording order, the processes and the readings
// of a history.
func TestHistory(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("returns an empty history", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, history.New().Events(), "no event")
		})
	})

	t.Run("Invoke", func(t *testing.T) {
		t.Parallel()

		t.Run("records an invocation at the next index with its call, client, process, operation, args and keys",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				recordOK(h, 4, write, writeOne, nil, "x")
				h.Invoke(4, read, nil, "x", "y")
				want := history.Event{
					Index: 2, Kind: history.Invoke, Call: 2, Client: 4, Process: 0, Operation: read,
					Keys: []any{"x", "y"},
				}
				assert.Equal(t, h.Events()[2], want, "the invocation")
			})

		t.Run("keeps two maps of equal entries one key, whatever order their entries iterate in", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			for v := range 8 {
				recordOK(h, 0, write, []any{v + 1}, nil, numbered())
			}
			recordOK(h, 0, read, nil, 0, numbered())
			got := detailOf(h, register)
			assert.Equal(t, got[partitionsField], any(1), "every call is in the partition of the one key")
		})

		t.Run("records into the zero History as into a new one", func(t *testing.T) {
			t.Parallel()
			var h history.History
			h.Invoke(0, write, writeOne, "x").OK(nil)
			assert.Length(t, h.Events(), 2, "the invocation and its completion")
		})

		t.Run("numbers the processes in the order of each client's first invocation", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 5, write, writeOne, nil)
			recordOK(h, 2, write, writeOne, nil)
			recordOK(h, 5, read, nil, 1)
			assert.Equal(t, processes(h), []int{0, 0, 1, 1, 0, 0}, "client 5 on process 0 and client 2 on 1")
		})

		t.Run("keeps a client on its process after a call that completes as OK or Fail", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, write, writeOne).OK(nil)
			h.Invoke(0, write, writeOne).Fail(errRefused)
			h.Invoke(0, read, nil).OK(1)
			assert.Equal(t, processes(h), []int{0, 0, 0, 0, 0, 0}, "one process")
		})

		t.Run("moves a client to a new process after a call whose outcome is unknown", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, write, writeOne).Unknown(errRefused)
			recordOK(h, 1, read, nil, 1)
			recordOK(h, 0, read, nil, 1)
			assert.Equal(t, processes(h), []int{0, 0, 1, 1, 2, 2}, "client 0 continues on process 2")
		})

		t.Run("panics for a client whose call is open", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 1, write, writeOne, nil)
			h.Invoke(0, write, writeOne)
			got := assert.Panics(t, func() { h.Invoke(0, read, nil) }, "a second open call of client 0")
			assert.Equal(t, got, any(`history: Invoke(0, "read") while call 2 of client 0 is open`),
				"the panic names the call and the open one")
			assert.Length(t, h.Events(), 3, "the history records nothing of the refused call")
		})

		t.Run("panics for a key that no typed literal states", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			got := assert.Panics(t, func() { h.Invoke(0, read, nil, "x", make(chan int)) }, "a channel as a key")
			assert.Equal(
				t,
				got,
				any(`history: Invoke(0, "read") states the key chan int, which no typed literal states`),
				"the panic names the call and the key's type",
			)
			assert.Empty(t, h.Events(), "the history records nothing of the refused call")
		})

		t.Run("takes two keys of equal typed literals as one key", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil, 1)
			recordOK(h, 1, read, nil, 0, int64(1))
			got := detailOf(h, register)
			assert.Equal(t, got[partitionsField], any(1), "1 and int64(1) are one key, so the read follows the write")
			assert.Equal(t, got[partitionField], any([]any{1}), "the partition states the key as declared first")
		})

		t.Run("takes two keys of different typed literals as two keys", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil, 1)
			recordOK(h, 1, read, nil, 0, 1.0)
			assert.Nil(t, detailOf(h, register), "1 and 1.0 are two keys, so the read starts from the initial state")
		})
	})

	t.Run("Events", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that a later call does not change", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil)
			got := h.Events()
			got[0].Client = 9
			recordOK(h, 0, read, nil, 1)
			assert.Length(t, got, 2, "the events recorded before the call")
			assert.Equal(t, h.Events()[0].Client, 0, "the history keeps its own event")
		})

		t.Run("returns every event recorded before the call while eight clients record", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			var clients sync.WaitGroup
			for client := range recordingClients {
				clients.Go(func() {
					for range recordedCalls {
						recordOK(h, client, write, writeOne, nil, "x")
					}
				})
			}
			done := make(chan struct{})
			go func() {
				clients.Wait()
				close(done)
			}()
			for reading := true; reading; {
				select {
				case <-done:
					reading = false
				default:
				}
				expectWhole(t, h.Events())
			}
			assert.Length(t, h.Events(), 2*recordingClients*recordedCalls, "every call's two events")
		})
	})
}

// processes returns the process of each event of h, in recording order.
func processes(h *history.History) []int {
	events := h.Events()
	out := make([]int, 0, len(events))
	for _, e := range events {
		out = append(out, e.Process)
	}
	return out
}

// expectWhole checks that events are a reading without a gap: each event's
// index is its position, and each completion follows the invocation of its
// call by the same client.
func expectWhole(tb testing.TB, events []history.Event) {
	tb.Helper()
	for i, e := range events {
		if e.Index != i {
			tb.Fatalf("the event at %d states the index %d", i, e.Index)
		}
		if e.Kind == history.Invoke {
			continue
		}
		if invocation := events[e.Call]; invocation.Kind != history.Invoke || invocation.Client != e.Client {
			tb.Fatalf("the completion at %d names the event %+v as its invocation", i, invocation)
		}
	}
}

// recorded is a history of one call, which the measurement of Events reads.
var recorded = func() *history.History {
	h := history.New()
	recordOK(h, 0, write, writeOne, nil, "x")
	return h
}()

// historyAllocs are the cases of the allocation ceilings of a history.
var historyAllocs = []alloctest.Case{
	{Name: "New", Call: func(assert.TB) { _ = history.New() }, Allocs: newAllocs},
	{Name: "Invoke", Call: func(assert.TB) { history.New().Invoke(0, write, writeOne, "x") }, Allocs: invokeAllocs},
	{Name: "Events", Call: func(assert.TB) { _ = recorded.Events() }, Allocs: eventsAllocs},
}

// TestHistoryAllocs checks the allocation ceilings of a history.
func TestHistoryAllocs(t *testing.T) {
	alloctest.Check(t, historyAllocs)
}

// BenchmarkHistory measures a history under its ceilings.
func BenchmarkHistory(b *testing.B) {
	for _, c := range historyAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
