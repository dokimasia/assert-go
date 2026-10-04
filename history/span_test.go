// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
)

// spanJSONAllocs are the allocations of the JSON form of a known span of
// one int argument and an int output, measured.
const spanJSONAllocs = 20

// TestSpan checks the JSON form of a call in the record of a check, which
// the definition fixes.
func TestSpan(t *testing.T) {
	t.Parallel()

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give history.Span
			want string
		}{
			{
				name: "returns a known call with its completion and its output",
				give: history.Span{
					Call: 2, Completion: 5, Process: 1,
					Op: history.Op{Operation: write, Args: []any{1}, Known: true, Output: "done"},
				},
				want: `{"call":2,"completion":5,"process":1,"operation":"write","args":[{"type":"int","value":1}],` +
					`"output":{"type":"string","value":"done"}}`,
			},
			{
				name: "returns a known call whose output is null",
				give: history.Span{Call: 0, Completion: 1, Op: history.Op{Operation: write, Known: true}},
				want: `{"call":0,"completion":1,"process":0,"operation":"write","args":[],"output":{"type":"null"}}`,
			},
			{
				name: "returns a call whose outcome is unknown with its completion and without an output",
				give: history.Span{
					Call:       2,
					Completion: 3,
					Process:    1,
					Op:         history.Op{Operation: write, Args: []any{1}},
				},
				want: `{"call":2,"completion":3,"process":1,"operation":"write","args":[{"type":"int","value":1}]}`,
			},
			{
				name: "returns a pending call without a completion and without an output",
				give: history.Span{Call: 0, Completion: -1, Process: 0, Op: history.Op{Operation: read}},
				want: `{"call":0,"process":0,"operation":"read","args":[]}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.give.MarshalJSON()
				assert.NoError(t, err, "a span marshals")
				assert.Equal(t, string(got), tt.want, "the history's JSON form of a call")
			})
		}
	})
}

// spanAllocs are the cases of the allocation ceiling of a span's JSON form.
var spanAllocs = []alloctest.Case{
	{
		Name: "MarshalJSON",
		Call: func(assert.TB) {
			s := history.Span{
				Call:       2,
				Completion: 5,
				Op:         history.Op{Operation: write, Args: []any{1}, Known: true, Output: 7},
			}
			_, _ = s.MarshalJSON()
		},
		Allocs: spanJSONAllocs,
	},
}

// TestSpanAllocs checks the allocation ceiling of a span's JSON form.
func TestSpanAllocs(t *testing.T) {
	alloctest.Check(t, spanAllocs)
}

// BenchmarkSpan measures the JSON form of a span under its ceiling.
func BenchmarkSpan(b *testing.B) {
	for _, c := range spanAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
