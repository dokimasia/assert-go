// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
)

// longFork returns the history of a long fork: two transactions append to x
// and to y, and two readers observe the appends in opposite orders. Its
// cycle has two read-write dependencies, and no two of them are adjacent.
func longFork() *history.History {
	h := history.New()
	transact(h, 0, history.OK, appendOf("x", 1))
	transact(h, 1, history.OK, appendOf("y", 1))
	transact(h, 2, history.OK, readOf("x", 1), readOf("y"))
	transact(h, 3, history.OK, readOf("x"), readOf("y", 1))
	return h
}

// TestCycle checks the five kinds of cycle, the order in which the search
// finds them, and the record of each.
func TestCycle(t *testing.T) {
	t.Parallel()

	t.Run("Serializable", func(t *testing.T) {
		t.Parallel()

		t.Run("reports a cycle of write-write dependencies as G0", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1), appendOf("y", 2))
			transact(h, 1, history.OK, appendOf("x", 2), appendOf("y", 1))
			transact(h, 2, history.OK, readOf("x", 1, 2), readOf("y", 1, 2))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[kindsField], any([]history.Anomaly{history.G0}), "a cycle of ww edges alone")
			assert.Equal(t, got[explanationField], any([]history.Evidence{
				history.Edge{From: 0, To: 2, Relation: history.WW, Key: "x", Value: 1, Next: 2},
				history.Edge{From: 2, To: 0, Relation: history.WW, Key: "y", Value: 1, Next: 2},
			}), "x orders call 0 first, and y call 2 first")
		})

		t.Run("reports a cycle of write-read dependencies as G1c", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1), readOf("y", 1))
			transact(h, 1, history.OK, readOf("x", 1), appendOf("y", 1))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[anomalyField], any(history.G1c), "each transaction read the other's append")
			assert.Equal(t, got[cycleField], any([]history.Link{
				{Call: 0, Relations: []history.Relation{history.WR}},
				{Call: 2, Relations: []history.Relation{history.WR}},
			}), "two wr edges")
		})

		t.Run("reports a cycle with one read-write dependency as G-single, and as G2", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, readOf("x"), readOf("y", 1))
			transact(h, 1, history.OK, appendOf("x", 1), appendOf("y", 1))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got, map[string]any{
				anomalyField: history.GSingle,
				kindsField:   []history.Anomaly{history.GSingle, history.G2},
				transactionsField: []history.Transaction{
					committedAt(0, 0, readOf("x"), readOf("y", 1)),
					committedAt(2, 1, appendOf("x", 1), appendOf("y", 1)),
				},
				cycleField: []history.Link{
					{Call: 0, Relations: []history.Relation{history.RW}},
					{Call: 2, Relations: []history.Relation{history.WR}},
				},
				explanationField: []history.Evidence{
					history.Edge{From: 0, To: 2, Relation: history.RW, Key: "x", Next: 1, Empty: true},
					history.Edge{From: 2, To: 0, Relation: history.WR, Key: "y", Value: 1},
				},
			}, "the detail of a read skew")
		})

		t.Run(
			"reports a cycle of two read-write dependencies that are not adjacent as G-nonadjacent",
			func(t *testing.T) {
				t.Parallel()
				got := isolationOf(history.Serializable, longFork())
				assert.Equal(t, got[kindsField], any([]history.Anomaly{history.GNonadjacent, history.G2}),
					"a long fork is G-nonadjacent and G2")
				assert.Equal(t, got[explanationField], any([]history.Evidence{
					history.Edge{From: 4, To: 2, Relation: history.RW, Key: "y", Next: 1, Empty: true},
					history.Edge{From: 2, To: 6, Relation: history.WR, Key: "y", Value: 1},
					history.Edge{From: 6, To: 0, Relation: history.RW, Key: "x", Next: 1, Empty: true},
					history.Edge{From: 0, To: 4, Relation: history.WR, Key: "x", Value: 1},
				}), "the cycle starts at its first rw edge")
			},
		)

		t.Run("reports a write skew as G2", func(t *testing.T) {
			t.Parallel()
			got := isolationOf(history.Serializable, writeSkew())
			assert.Equal(t, got[cycleField], any([]history.Link{
				{Call: 0, Relations: []history.Relation{history.RW}},
				{Call: 2, Relations: []history.Relation{history.RW}},
			}), "two adjacent rw edges")
		})

		t.Run("tries the components in the order of their lowest transactions", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, readOf("x"), appendOf("y", 1), readOf("z"))
			transact(h, 1, history.OK, readOf("u"), appendOf("v", 1), appendOf("z", 1))
			transact(h, 2, history.OK, readOf("v"), appendOf("u", 1))
			transact(h, 3, history.OK, readOf("y"), appendOf("x", 1))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[cycleField], any([]history.Link{
				{Call: 0, Relations: []history.Relation{history.RW}},
				{Call: 6, Relations: []history.Relation{history.RW}},
			}), "the write skew of calls 0 and 6, whose component Tarjan's search closes second")
		})

		t.Run("reduces a walk that passes a transaction twice, and drops a part with adjacent read-write edges",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				transact(h, 0, history.OK, readOf("k1"), appendOf("k6", 2))
				transact(h, 0, history.OK, appendOf("k1", 1), appendOf("k2", 1))
				transact(h, 0, history.OK, appendOf("k2", 2), readOf("k3"), appendOf("k5", 1), appendOf("k6", 1))
				transact(h, 0, history.OK, appendOf("k3", 1), appendOf("k4", 1))
				transact(h, 0, history.OK, appendOf("k4", 2), readOf("k5"))
				transact(h, 0, history.OK, readOf("k1", 1), readOf("k2", 1, 2), readOf("k3", 1), readOf("k4", 1, 2),
					readOf("k5", 1), readOf("k6", 1, 2))
				got := isolationOf(history.Serializable, h)
				assert.Equal(t, got[kindsField], any([]history.Anomaly{history.GSingle, history.G2}),
					"each walk reduces to a cycle with one read-write edge")
			})

		t.Run("passes a history whose dependencies are acyclic", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1))
			transact(h, 1, history.OK, readOf("x", 1), appendOf("x", 2))
			transact(h, 2, history.OK, readOf("x", 1, 2))
			assert.Nil(t, isolationOf(history.Serializable, h), "a serial history")
		})
	})

	t.Run("HasSnapshotIsolation", func(t *testing.T) {
		t.Parallel()

		t.Run("passes a write skew, whose two rw edges are adjacent", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, isolationOf(history.HasSnapshotIsolation, writeSkew()), "snapshot isolation permits it")
		})

		t.Run("reports a long fork as G-nonadjacent", func(t *testing.T) {
			t.Parallel()
			got := isolationOf(history.HasSnapshotIsolation, longFork())
			assert.Equal(t, got[kindsField], any([]history.Anomaly{history.GNonadjacent}),
				"snapshot isolation forbids every kind but G2")
		})
	})
}
