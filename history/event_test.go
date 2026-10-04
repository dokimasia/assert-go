// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
)

// errRefused is the error of a call that took no effect.
var errRefused = errors.New("history_test: the connection is refused")

// The allocations of the JSON form of an event, measured.
const (
	// invocationJSONAllocs are the allocations of MarshalJSON for an
	// invocation of one int argument on one string key.
	invocationJSONAllocs = 21
	// completionJSONAllocs are the allocations of MarshalJSON for an ok
	// completion with an int output.
	completionJSONAllocs = 11
)

// TestEvent checks the JSON form of an event, which the definition fixes.
func TestEvent(t *testing.T) {
	t.Parallel()

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give history.Event
			want string
		}{
			{
				name: "returns an invocation with its operation, and its args and keys as typed literals",
				give: history.Event{
					Index: 2, Kind: history.Invoke, Call: 2, Client: 3, Process: 1, Operation: write,
					Args: []any{1}, Keys: []any{"x"},
				},
				want: `{"index":2,"kind":"invoke","call":2,"client":3,"process":1,"operation":"write",` +
					`"args":[{"type":"int","value":1}],"keys":[{"type":"string","value":"x"}]}`,
			},
			{
				name: "returns an invocation without args or keys with two empty lists",
				give: history.Event{Kind: history.Invoke, Operation: read},
				want: `{"index":0,"kind":"invoke","call":0,"client":0,"process":0,"operation":"read",` +
					`"args":[],"keys":[]}`,
			},
			{
				name: "returns an ok completion with its output as a typed literal",
				give: history.Event{Index: 3, Kind: history.OK, Call: 2, Client: 3, Process: 1, Output: 7},
				want: `{"index":3,"kind":"ok","call":2,"client":3,"process":1,"output":{"type":"int","value":7}}`,
			},
			{
				name: "returns a fail completion with the text of its error",
				give: history.Event{Index: 3, Kind: history.Fail, Call: 2, Error: errRefused},
				want: `{"index":3,"kind":"fail","call":2,"client":0,"process":0,` +
					`"error":"history_test: the connection is refused"}`,
			},
			{
				name: "returns an unknown completion without an error with the empty text",
				give: history.Event{Index: 1, Kind: history.Unknown},
				want: `{"index":1,"kind":"unknown","call":0,"client":0,"process":0,"error":""}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.give.MarshalJSON()
				assert.NoError(t, err, "an event marshals")
				assert.Equal(t, string(got), tt.want, "the history's JSON form")
			})
		}
	})
}

// eventAllocs are the cases of the allocation ceilings of an event's JSON
// form.
var eventAllocs = []alloctest.Case{
	{
		Name: "MarshalJSON/invocation",
		Call: func(assert.TB) {
			e := history.Event{Kind: history.Invoke, Operation: write, Args: []any{1}, Keys: []any{"x"}}
			_, _ = e.MarshalJSON()
		},
		Allocs: invocationJSONAllocs,
	},
	{
		Name: "MarshalJSON/completion",
		Call: func(assert.TB) {
			e := history.Event{Index: 1, Kind: history.OK, Output: 7}
			_, _ = e.MarshalJSON()
		},
		Allocs: completionJSONAllocs,
	},
}

// TestEventAllocs checks the allocation ceilings of an event's JSON form.
func TestEventAllocs(t *testing.T) {
	alloctest.Check(t, eventAllocs)
}

// BenchmarkEvent measures the JSON form of an event under its ceilings.
func BenchmarkEvent(b *testing.B) {
	for _, c := range eventAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
