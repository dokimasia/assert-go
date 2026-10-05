// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
)

// TestDerivation checks which transactions are committed, the version order
// of a key, and the dependencies that the reads reveal, through the cycles
// that a check reports.
func TestDerivation(t *testing.T) {
	t.Parallel()

	t.Run("Serializable", func(t *testing.T) {
		t.Parallel()

		t.Run("commits a transaction of unknown outcome whose append a committed read observed", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.Unknown, appendOf("x", 1), appendOf("y", 2))
			transact(h, 1, history.OK, readOf("x", 1), appendOf("y", 1))
			transact(h, 2, history.OK, readOf("y", 1, 2))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[cycleField], any([]history.Link{
				{Call: 0, Relations: []history.Relation{history.WR}},
				{Call: 2, Relations: []history.Relation{history.WW}},
			}), "call 1 read call 0's append, which follows call 1's in y")
			assert.Equal(t, got[transactionsField], any([]history.Transaction{
				{Call: 0, Completion: 1, Kind: history.Unknown, Args: []any{appendOf("x", 1), appendOf("y", 2)}},
				committedAt(2, 1, readOf("x", 1), appendOf("y", 1)),
			}), "the unknown transaction, then the transaction that observed it")
		})

		t.Run("leaves out a transaction of unknown outcome whose append no read observed", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.Unknown, appendOf("x", 1), appendOf("y", 2))
			transact(h, 1, history.OK, readOf("x"), appendOf("y", 1))
			transact(h, 2, history.OK, readOf("y", 1))
			assert.Nil(t, isolationOf(history.Serializable, h), "no dependency involves the unknown transaction")
		})

		t.Run("leaves out a pending transaction whose append no read observed", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.Invoke, appendOf("x", 1))
			transact(h, 1, history.OK, readOf("x"))
			assert.Nil(t, isolationOf(history.Serializable, h), "no dependency involves the pending transaction")
		})

		t.Run("places a committed append that no read observed after the version order", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 2), appendOf("y", 1))
			transact(h, 1, history.OK, appendOf("x", 1), readOf("y", 1))
			transact(h, 2, history.OK, readOf("x", 1))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[explanationField], any([]history.Evidence{
				history.Edge{From: 0, To: 2, Relation: history.WR, Key: "y", Value: 1},
				history.Edge{From: 2, To: 0, Relation: history.WW, Key: "x", Value: 1, Next: 2},
			}), "call 0's unobserved append of 2 follows call 2's append of 1")
		})

		t.Run("explains a dependency by the first key that reveals it", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1), appendOf("y", 1), appendOf("z", 2))
			transact(h, 1, history.OK, appendOf("x", 2), appendOf("y", 2), appendOf("z", 1))
			transact(h, 2, history.OK, readOf("x", 1, 2), readOf("y", 1, 2), readOf("z", 1, 2))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[explanationField], any([]history.Evidence{
				history.Edge{From: 0, To: 2, Relation: history.WW, Key: "x", Value: 1, Next: 2},
				history.Edge{From: 2, To: 0, Relation: history.WW, Key: "z", Value: 1, Next: 2},
			}), "x and y both order call 0 before call 2, and x comes first")
		})

		t.Run("derives no dependency of a transaction on itself", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1), readOf("x", 1))
			transact(h, 1, history.OK, readOf("x", 1), appendOf("x", 2))
			transact(h, 2, history.OK, readOf("x", 1, 2))
			assert.Nil(t, isolationOf(history.Serializable, h), "the reads of own appends close no cycle")
		})

		t.Run("derives no dependency from a key whose reads are no prefixes of one list", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1), readOf("y"))
			transact(h, 1, history.OK, appendOf("x", 2), appendOf("y", 1))
			transact(h, 2, history.OK, readOf("x", 1))
			transact(h, 3, history.OK, readOf("x", 2))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[kindsField], any([]history.Anomaly{history.IncompatibleOrder}),
				"x gives no edge that would close a cycle with y's")
		})
	})
}
