// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/matcher"
)

// TestRender checks the sentence that the text writer writes for the record
// of a failing check.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("sentence", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the outcome, the partition, the counts and the frontier of a violated check", func(t *testing.T) {
			t.Parallel()
			want := `the register is linearizable: violated in the partition of "x"` +
				"\n    steps 2, partitions 1, calls 2, concurrency 1" +
				"\n    linearized: call 0 write(1) → <nil>" +
				"\n    states: 1" +
				"\n    rejected: call 2 read() → 0"
			assert.Equal(t, sentenceOf(violatedRead(), register), want, "the sentence of the violated read")
		})

		t.Run("writes the limit of an undecided check, and none for no rejected call", func(t *testing.T) {
			t.Parallel()
			want := `the register is linearizable: undecided in the partition of "x", at the steps limit` +
				"\n    steps 1, partitions 1, calls 2, concurrency 1" +
				"\n    linearized: call 0 write(1) → <nil>" +
				"\n    states: 1" +
				"\n    rejected: none"
			assert.Equal(t, sentenceOf(violatedRead(), register, history.Budget(1)), want,
				"the sentence of the stopped search")
		})

		t.Run("writes every key for a partition of every key, and an unknown outcome for a call that is not known",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				h.Invoke(0, write, writeOne).Unknown(errRefused)
				recordOK(h, 1, read, nil, 7)
				want := "the register is linearizable: violated in the partition of every key" +
					"\n    steps 3, partitions 1, calls 2, concurrency 2" +
					"\n    linearized: call 0 write(1), outcome unknown" +
					"\n    states: 1" +
					"\n    rejected: call 2 read() → 7"
				assert.Equal(t, sentenceOf(h, register), want, "the sentence of the unknown write")
			})

		t.Run("separates the keys, the calls, their args and the states of a list", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, "swap", []any{1, 2}, nil, "y", "x")
			recordOK(h, 0, "swap", []any{3, 4}, nil, "x")
			m := history.Model[int]{
				Init: register.Init,
				Step: func(_ int, op history.Op) []int {
					if op.Args[0] == 1 {
						return []int{5, 6}
					}
					return nil
				},
			}
			want := `the register is linearizable: violated in the partition of "y", "x"` +
				"\n    steps 3, partitions 1, calls 2, concurrency 1" +
				"\n    linearized: call 0 swap(1, 2) → <nil>" +
				"\n    states: 5, 6" +
				"\n    rejected: call 2 swap(3, 4) → <nil>"
			assert.Equal(t, sentenceOf(h, m), want, "the lists of the sentence")
		})

		t.Run("writes the record of linearizable that states no field", func(t *testing.T) {
			t.Parallel()
			got := matcher.Render(assert.Failure{Assertion: "linearizable", Contract: contract})
			assert.HasPrefix(t, got, contract+": <nil> in the partition of every key", "the contract, and no value")
		})
	})

	t.Run("isolationSentence", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the anomaly, every kind, and a line for each edge of the cycle", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, readOf("x"), readOf("y", 1))
			transact(h, 1, history.OK, appendOf("x", 1), appendOf("y", 1))
			rec := assert.NewRecorder()
			history.Serializable(rec, h, isolationContract)
			want := isolationContract + ": G-single" +
				"\n    kinds: G-single, G2" +
				"\n    call 0 -rw-> call 2: call 0 read [] from \"x\", and call 2 appended 1 to it" +
				"\n    call 2 -wr-> call 0: call 0 read \"y\" ending in 1"
			assert.Equal(t, matcher.Render(rec.Failures()[0]), want, "the sentence of the read skew")
		})

		tests := []struct {
			name string
			give history.Evidence
			want string
		}{
			{
				name: "writes the two values of a write-write edge",
				give: history.Edge{From: 0, To: 2, Relation: history.WW, Key: "x", Value: 1, Next: 2},
				want: `call 0 -ww-> call 2: call 2 appended 2 to "x" after 1`,
			},
			{
				name: "writes the last value that the reader read of a write-read edge",
				give: history.Edge{From: 0, To: 2, Relation: history.WR, Key: "x", Value: 1},
				want: `call 0 -wr-> call 2: call 2 read "x" ending in 1`,
			},
			{
				name: "writes the last value read and the next value of a read-write edge",
				give: history.Edge{From: 0, To: 2, Relation: history.RW, Key: "x", Value: 1, Next: 2},
				want: `call 0 -rw-> call 2: call 0 read "x" ending in 1, and call 2 appended 2 after it`,
			},
			{
				name: "writes the empty read of a read-write edge",
				give: history.Edge{From: 0, To: 2, Relation: history.RW, Key: "x", Next: 2, Empty: true},
				want: `call 0 -rw-> call 2: call 0 read [] from "x", and call 2 appended 2 to it`,
			},
			{
				name: "writes the read and the value of a garbage read",
				give: history.Observation{
					Anomaly: history.GarbageRead, Calls: []int{0}, Key: "x", Reads: [][]any{{7}}, Value: 7,
				},
				want: `call 0 read [7] from "x", and no transaction appended 7`,
			},
			{
				name: "writes the read and the repeated value of a duplicate append",
				give: history.Observation{
					Anomaly: history.DuplicateAppend, Calls: []int{2}, Key: "x", Reads: [][]any{{1, 2, 2}}, Value: 2,
				},
				want: `call 2 read [1, 2, 2] from "x", which contains 2 twice`,
			},
			{
				name: "writes the whole list that an internally inconsistent read contradicts",
				give: history.Observation{
					Anomaly: history.InternalInconsistency, Calls: []int{2}, Key: "x", Reads: [][]any{{1}},
					Expected: []any{}, Whole: true,
				},
				want: `call 2 read [1] from "x", and it knew the list was []`,
			},
			{
				name: "writes the end of the list that an internally inconsistent read lacks",
				give: history.Observation{
					Anomaly: history.InternalInconsistency, Calls: []int{0}, Key: "x", Reads: [][]any{{}},
					Expected: []any{1},
				},
				want: `call 0 read [] from "x", and it knew the list ended with [1]`,
			},
			{
				name: "writes the later append that an internally inconsistent read contains",
				give: history.Observation{
					Anomaly: history.InternalInconsistency, Calls: []int{0}, Key: "x", Reads: [][]any{{1}},
					Expected: []any{}, Future: 1, HasFuture: true,
				},
				want: `call 0 read [1] from "x", and 1 is a value that call 0 appends later`,
			},
			{
				name: "writes the two reads of an incompatible order",
				give: history.Observation{
					Anomaly: history.IncompatibleOrder, Calls: []int{4, 6}, Key: "x", Reads: [][]any{{1}, {2}},
				},
				want: `calls 4 and 6 read [1] and [2] from "x", and neither is a prefix of the other`,
			},
			{
				name: "writes the value and its appender of an aborted read",
				give: history.Observation{Anomaly: history.AbortedRead, Calls: []int{2}, Key: "x", Value: 1},
				want: `call 2 read 1 from "x", which call 0 appended and aborted`,
			},
			{
				name: "writes the value, its appender and the appender's next value of an intermediate read",
				give: history.Observation{
					Anomaly: history.IntermediateRead, Calls: []int{2}, Key: "x", Value: 1, Next: 2,
				},
				want: `call 2 read "x" ending in 1, which call 0 followed with 2`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := matcher.Render(assert.Failure{
					Assertion: "serializable", Contract: isolationContract,
					Detail: map[string]any{
						anomalyField: history.G2, kindsField: []history.Anomaly{history.G2},
						explanationField: []history.Evidence{tt.give},
					},
				})
				assert.Equal(t, got, isolationContract+": G2\n    kinds: G2\n    "+tt.want, "the line of the evidence")
			})
		}

		t.Run("writes the record of serializable that states no field", func(t *testing.T) {
			t.Parallel()
			got := matcher.Render(assert.Failure{Assertion: "serializable", Contract: isolationContract})
			assert.Equal(t, got, isolationContract+": <nil>\n    kinds: ", "the contract, and no value")
		})
	})
}

// sentenceOf checks h against m on a recorder under opts, and returns the
// sentence of the record of the check.
func sentenceOf(h *history.History, m history.Model[int], opts ...history.Option) string {
	rec := assert.NewRecorder()
	history.Linearizable(rec, h, m, contract, opts...)
	return matcher.Render(rec.Failures()[0])
}
