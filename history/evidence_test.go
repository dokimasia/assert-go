// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
)

// The allocations of the JSON form of each kind of evidence, measured.
const (
	// edgeJSONAllocs are the allocations of the JSON form of rwEdge.
	edgeJSONAllocs = 24
	// observationJSONAllocs are the allocations of the JSON form of
	// inconsistentRead.
	observationJSONAllocs = 39
)

// The typed literals of the keys, the values and the lists of the tests of
// the JSON form of evidence.
const (
	keyX       = `{"type":"string","value":"x"}`
	oneLiteral = `{"type":"int","value":1}`
	twoLiteral = `{"type":"int","value":2}`
	oneList    = `{"type":"list","of":"int","value":[1]}`
	twoList    = `{"type":"list","of":"int","value":[2]}`
	emptyList  = `{"type":"list","items":[]}`
)

// rwEdge is the read-write edge of call 0 that read [1] from x, on call 2,
// which appended 2 after it.
var rwEdge = history.Edge{From: 0, To: 2, Relation: history.RW, Key: "x", Value: 1, Next: 2}

// rwEdgeJSON is the JSON form of rwEdge.
const rwEdgeJSON = `{"from":0,"to":2,"relation":"rw","key":` + keyX + `,"value":` + oneLiteral +
	`,"next":` + twoLiteral + `}`

// inconsistentRead is the observation of call 4's read of [1] from x, which
// knew the whole list [2] and appends 1 to x later.
var inconsistentRead = history.Observation{
	Anomaly: history.InternalInconsistency, Calls: []int{4}, Key: "x", Reads: [][]any{{1}},
	Expected: []any{2}, Whole: true, Future: 1, HasFuture: true,
}

// TestEvidence checks the JSON form of each kind of evidence and of a link,
// as the explanation and the cycle of a record state them.
func TestEvidence(t *testing.T) {
	t.Parallel()

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give history.Evidence
			want string
		}{
			{
				name: "states the value of the earlier call and the next value of a write-write edge",
				give: history.Edge{From: 0, To: 2, Relation: history.WW, Key: "x", Value: 1, Next: 2},
				want: `{"from":0,"to":2,"relation":"ww","key":` + keyX + `,"value":` + oneLiteral +
					`,"next":` + twoLiteral + `}`,
			},
			{
				name: "states the last value read and no next value of a write-read edge",
				give: history.Edge{From: 0, To: 2, Relation: history.WR, Key: "x", Value: 1},
				want: `{"from":0,"to":2,"relation":"wr","key":` + keyX + `,"value":` + oneLiteral + `}`,
			},
			{
				name: "states the last value read and the next value of a read-write edge",
				give: rwEdge,
				want: rwEdgeJSON,
			},
			{
				name: "states null for the value of a read-write edge of an empty read",
				give: history.Edge{From: 0, To: 2, Relation: history.RW, Key: "x", Next: 2, Empty: true},
				want: `{"from":0,"to":2,"relation":"rw","key":` + keyX + `,"value":null,"next":` + twoLiteral + `}`,
			},
			{
				name: "states the null value that a read-write edge read last as a typed literal",
				give: history.Edge{From: 0, To: 2, Relation: history.RW, Key: "x", Next: 2},
				want: `{"from":0,"to":2,"relation":"rw","key":` + keyX + `,"value":{"type":"null"},"next":` +
					twoLiteral + `}`,
			},
			{
				name: "states the call, the key, the read and the value of a garbage read",
				give: history.Observation{
					Anomaly: history.GarbageRead, Calls: []int{4}, Key: "x", Reads: [][]any{{1}}, Value: 1,
				},
				want: `{"call":4,"key":` + keyX + `,"read":` + oneList + `,"value":` + oneLiteral + `}`,
			},
			{
				name: "states the call, the key, the read and the value of a duplicate append",
				give: history.Observation{
					Anomaly: history.DuplicateAppend, Calls: []int{4}, Key: "x", Reads: [][]any{{1}}, Value: 1,
				},
				want: `{"call":4,"key":` + keyX + `,"read":` + oneList + `,"value":` + oneLiteral + `}`,
			},
			{
				name: "states what an internally inconsistent read expected, and its future append",
				give: inconsistentRead,
				want: `{"call":4,"key":` + keyX + `,"read":` + oneList + `,"expected":` + twoList +
					`,"whole":true,"future":` + oneLiteral + `}`,
			},
			{
				name: "states null for the future of an internally inconsistent read without one",
				give: history.Observation{
					Anomaly: history.InternalInconsistency, Calls: []int{4}, Key: "x", Reads: [][]any{{1}},
					Expected: []any{},
				},
				want: `{"call":4,"key":` + keyX + `,"read":` + oneList + `,"expected":` + emptyList +
					`,"whole":false,"future":null}`,
			},
			{
				name: "states the two calls, the key and the two reads of an incompatible order",
				give: history.Observation{
					Anomaly: history.IncompatibleOrder, Calls: []int{4, 6}, Key: "x", Reads: [][]any{{1}, {2}},
				},
				want: `{"calls":[4,6],"key":` + keyX + `,"reads":[` + oneList + `,` + twoList + `]}`,
			},
			{
				name: "states the call, the key, the value and its appender of an aborted read",
				give: history.Observation{
					Anomaly:  history.AbortedRead,
					Calls:    []int{4},
					Key:      "x",
					Value:    1,
					Appender: 2,
				},
				want: `{"call":4,"key":` + keyX + `,"value":` + oneLiteral + `,"appender":2}`,
			},
			{
				name: "states the appender's next value of an intermediate read",
				give: history.Observation{
					Anomaly: history.IntermediateRead, Calls: []int{4}, Key: "x", Value: 1, Appender: 2, Next: 2,
				},
				want: `{"call":4,"key":` + keyX + `,"value":` + oneLiteral + `,"appender":2,"next":` + twoLiteral + `}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.give.MarshalJSON()
				assert.NoError(t, err, "the evidence marshals")
				assert.Equal(t, string(got), tt.want, "the entry of the explanation")
			})
		}
	})

	t.Run("Link", func(t *testing.T) {
		t.Parallel()

		t.Run("states the call and the spellings of the relations", func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(history.Link{Call: 2, Relations: []history.Relation{history.WW, history.RW}})
			assert.NoError(t, err, "a link marshals")
			assert.Equal(t, string(got), `{"call":2,"relations":["ww","rw"]}`, "the link of a cycle")
		})
	})
}

// evidenceAllocs are the cases of the allocation ceilings of the JSON form
// of each kind of evidence.
var evidenceAllocs = []alloctest.Case{
	{Name: "Edge.MarshalJSON", Call: func(assert.TB) { _, _ = rwEdge.MarshalJSON() }, Allocs: edgeJSONAllocs},
	{
		Name:   "Observation.MarshalJSON",
		Call:   func(assert.TB) { _, _ = inconsistentRead.MarshalJSON() },
		Allocs: observationJSONAllocs,
	},
}

// TestEvidenceAllocs checks the allocation ceilings of the JSON form of each
// kind of evidence.
func TestEvidenceAllocs(t *testing.T) {
	alloctest.Check(t, evidenceAllocs)
}

// BenchmarkEvidence measures the JSON form of each kind of evidence under
// its ceiling.
func BenchmarkEvidence(b *testing.B) {
	for _, c := range evidenceAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
