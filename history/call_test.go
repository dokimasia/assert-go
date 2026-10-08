// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
)

// completionAllocs are the allocations of a first invocation without keys
// and its completion, with the history they record into, measured.
const completionAllocs = 9

// secondCompletion is the panic of a second completion of the call at index
// 0.
const secondCompletion = "history: call 0 completes a second time"

// TestCall checks the completions of a call.
func TestCall(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		t.Run("records an ok completion of the call with its output", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 1, write, writeOne, nil)
			h.Invoke(3, read, nil).OK(1)
			want := history.Event{Index: 3, Kind: history.OK, Call: 2, Client: 3, Process: 1, Output: 1}
			assert.Equal(t, h.Events()[3], want, "the completion names the invocation of its call")
		})

		t.Run("panics when the call has completed already", func(t *testing.T) {
			t.Parallel()
			c := history.New().Invoke(0, read, nil)
			c.Unknown(errRefused)
			got := assert.Panics(t, func() { c.OK(1) }, "a second completion")
			assert.Equal(t, got, any(secondCompletion), "the panic names the call")
		})

		t.Run("panics for a copy of a call that completed", func(t *testing.T) {
			t.Parallel()
			c := history.New().Invoke(0, read, nil)
			copied := c
			c.OK(1)
			got := assert.Panics(t, func() { copied.OK(1) }, "a completion of a copy")
			assert.Equal(t, got, any(secondCompletion), "the copy names the same call")
		})

		t.Run("panics for a call that completed after its client invoked again", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			c := h.Invoke(0, read, nil)
			c.OK(0)
			h.Invoke(0, read, nil)
			got := assert.Panics(t, func() { c.OK(0) }, "a completion of the earlier call")
			assert.Equal(t, got, any(secondCompletion), "the panic names the earlier call")
			assert.Length(t, h.Events(), 3, "the later call of the client is still open")
		})
	})

	t.Run("Fail", func(t *testing.T) {
		t.Parallel()

		t.Run("records a fail completion of the call with its error", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, write, writeOne).Fail(errRefused)
			want := history.Event{Index: 1, Kind: history.Fail, Call: 0, Error: errRefused}
			assert.Equal(t, h.Events()[1], want, "the completion and its error")
		})

		t.Run("panics when the call has completed already", func(t *testing.T) {
			t.Parallel()
			c := history.New().Invoke(0, write, writeOne)
			c.OK(nil)
			got := assert.Panics(t, func() { c.Fail(errRefused) }, "a second completion")
			assert.Equal(t, got, any(secondCompletion), "the panic names the call")
		})
	})

	t.Run("Unknown", func(t *testing.T) {
		t.Parallel()

		t.Run("records an unknown completion of the call with its error", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, write, writeOne).Unknown(errRefused)
			want := history.Event{Index: 1, Kind: history.Unknown, Call: 0, Error: errRefused}
			assert.Equal(t, h.Events()[1], want, "the completion and its error")
		})

		t.Run("panics when the call has completed already", func(t *testing.T) {
			t.Parallel()
			c := history.New().Invoke(0, write, writeOne)
			c.Fail(errRefused)
			got := assert.Panics(t, func() { c.Unknown(errRefused) }, "a second completion")
			assert.Equal(t, got, any(secondCompletion), "the panic names the call")
		})
	})
}

// callAllocs are the cases of the allocation ceilings of the completions of
// a call.
var callAllocs = []alloctest.Case{
	{Name: "OK", Call: func(assert.TB) { history.New().Invoke(0, read, nil).OK(nil) }, Allocs: completionAllocs},
	{
		Name:   "Fail",
		Call:   func(assert.TB) { history.New().Invoke(0, read, nil).Fail(errRefused) },
		Allocs: completionAllocs,
	},
	{
		Name:   "Unknown",
		Call:   func(assert.TB) { history.New().Invoke(0, read, nil).Unknown(errRefused) },
		Allocs: completionAllocs,
	},
}

// TestCallAllocs checks the allocation ceilings of the completions of a call.
func TestCallAllocs(t *testing.T) {
	alloctest.Check(t, callAllocs)
}

// BenchmarkCall measures the completions of a call under their ceilings.
func BenchmarkCall(b *testing.B) {
	for _, c := range callAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
