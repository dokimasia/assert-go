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
}

// sentenceOf checks h against m on a recorder under opts, and returns the
// sentence of the record of the check.
func sentenceOf(h *history.History, m history.Model[int], opts ...history.Option) string {
	rec := assert.NewRecorder()
	history.Linearizable(rec, h, m, contract, opts...)
	return matcher.Render(rec.Failures()[0])
}
