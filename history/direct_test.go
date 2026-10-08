// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"runtime"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestDirect checks the six anomalies that a check finds in the
// micro-operations of committed transactions, without the dependency graph.
func TestDirect(t *testing.T) {
	t.Parallel()

	t.Run("Serializable", func(t *testing.T) {
		t.Parallel()

		t.Run("reports a read of a value that no transaction appended as a garbage read", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, readOf("x", 7))
			assert.Equal(t, isolationOf(history.Serializable, h), map[string]any{
				anomalyField:      history.GarbageRead,
				kindsField:        []history.Anomaly{history.GarbageRead},
				transactionsField: []history.Transaction{committedAt(0, 0, readOf("x", 7))},
				cycleField:        nil,
				explanationField: []history.Evidence{history.Observation{
					Anomaly: history.GarbageRead, Calls: []int{0}, Key: "x", Reads: [][]any{{7}}, Value: 7,
				}},
			}, "the detail of a garbage read")
		})

		t.Run("reports a read that returned a value twice as a duplicate append, at the second", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1), appendOf("x", 2))
			transact(h, 1, history.OK, readOf("x", 1, 2, 2))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[explanationField], any([]history.Evidence{history.Observation{
				Anomaly: history.DuplicateAppend, Calls: []int{2}, Key: "x", Reads: [][]any{{1, 2, 2}}, Value: 2,
			}}), "the read and its duplicate value")
		})

		t.Run("reports a read that contains a later append of its transaction as an internal inconsistency",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				transact(h, 0, history.OK, readOf("x", 1), appendOf("x", 1))
				got := isolationOf(history.Serializable, h)
				assert.Equal(t, got[explanationField], any([]history.Evidence{history.Observation{
					Anomaly: history.InternalInconsistency, Calls: []int{0}, Key: "x", Reads: [][]any{{1}},
					Expected: []any{}, Future: 1, HasFuture: true,
				}}), "the read and the later append it contains")
			})

		t.Run("reports a read that lacks an earlier append of its transaction as an internal inconsistency",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				transact(h, 0, history.OK, appendOf("x", 1), readOf("x"))
				got := isolationOf(history.Serializable, h)
				assert.Equal(t, got[explanationField], any([]history.Evidence{history.Observation{
					Anomaly: history.InternalInconsistency, Calls: []int{0}, Key: "x", Reads: [][]any{{}},
					Expected: []any{1},
				}}), "the read and the end of the list it expected")
			})

		t.Run("reports a read that differs from an earlier read of its transaction as an internal inconsistency",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				transact(h, 0, history.OK, appendOf("x", 1))
				transact(h, 1, history.OK, readOf("x"), readOf("x", 1))
				got := isolationOf(history.Serializable, h)
				assert.Equal(t, got[explanationField], any([]history.Evidence{history.Observation{
					Anomaly: history.InternalInconsistency, Calls: []int{2}, Key: "x", Reads: [][]any{{1}},
					Expected: []any{}, Whole: true,
				}}), "the second read and the whole list it expected")
			})

		t.Run("accepts a read that ends with the earlier appends of its transaction", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1))
			transact(h, 1, history.OK, appendOf("x", 2), readOf("x", 1, 2), appendOf("x", 3))
			assert.Nil(t, isolationOf(history.Serializable, h), "the read observed the transaction's own append")
		})

		t.Run("reports two reads that are no prefixes of one list as an incompatible order", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1))
			transact(h, 1, history.OK, appendOf("x", 2))
			transact(h, 2, history.OK, readOf("x", 1))
			transact(h, 3, history.OK, readOf("x", 2))
			assert.Equal(t, isolationOf(history.Serializable, h), map[string]any{
				anomalyField: history.IncompatibleOrder,
				kindsField:   []history.Anomaly{history.IncompatibleOrder},
				transactionsField: []history.Transaction{
					committedAt(4, 2, readOf("x", 1)), committedAt(6, 3, readOf("x", 2)),
				},
				cycleField: nil,
				explanationField: []history.Evidence{history.Observation{
					Anomaly: history.IncompatibleOrder, Calls: []int{4, 6}, Key: "x", Reads: [][]any{{1}, {2}},
				}},
			}, "the detail of the two reads, the shorter first")
		})

		t.Run("reports a read of a value that an aborted transaction appended as an aborted read", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.Fail, appendOf("x", 1))
			transact(h, 1, history.OK, readOf("x", 1))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[transactionsField], any([]history.Transaction{
				committedAt(2, 1, readOf("x", 1)),
				{Call: 0, Completion: 1, Kind: history.Fail, Args: []any{appendOf("x", 1)}},
			}), "the reader, then the aborted appender")
			assert.Equal(t, got[explanationField], any([]history.Evidence{history.Observation{
				Anomaly: history.AbortedRead, Calls: []int{2}, Key: "x", Value: 1, Appender: 0,
			}}), "the value and its appender")
		})

		t.Run("reports a read that ends in a value its appender followed as an intermediate read", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1), appendOf("x", 2))
			transact(h, 1, history.OK, readOf("x", 1))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[explanationField], any([]history.Evidence{history.Observation{
				Anomaly: history.IntermediateRead, Calls: []int{2}, Key: "x", Value: 1, Appender: 0, Next: 2,
			}}), "the value, its appender and the appender's next value")
		})

		t.Run("accepts a read of a transaction's own append that the transaction follows", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1), readOf("x", 1), appendOf("x", 2))
			assert.Nil(t, isolationOf(history.Serializable, h), "a transaction's read of its own append")
		})

		t.Run("accepts a read of a value whose transaction's outcome is unknown", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.Unknown, appendOf("x", 1))
			transact(h, 1, history.OK, readOf("x", 1))
			assert.Nil(t, isolationOf(history.Serializable, h), "the unknown transaction committed")
		})

		t.Run("reports an internal inconsistency at a key that two maps of equal entries state", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf(numbered(), 1), readOf(numbered()))
			assert.Equal(t, isolationOf(history.Serializable, h)[anomalyField], any(history.InternalInconsistency),
				"the read of the one key lacks the transaction's append")
		})

		t.Run("reports the read list as read after the transaction's later appends", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			transact(h, 0, history.OK, appendOf("x", 1))
			transact(h, 1, history.OK, readOf("x", 1), appendOf("x", 2), appendOf("x", 3), readOf("x", 1, 3))
			got := isolationOf(history.Serializable, h)
			assert.Equal(t, got[explanationField], any([]history.Evidence{history.Observation{
				Anomaly: history.InternalInconsistency, Calls: []int{2}, Key: "x", Reads: [][]any{{1, 3}},
				Expected: []any{1, 2, 3}, Whole: true,
			}}), "the appends leave the first read's list as the read returned it")
		})
	})
}

// TestDirectBytesAllocs checks that the bytes that the check of one transaction's
// appends allocates grow with the number of the appends, and not with its
// square. It counts the allocations of the whole process, so it does not run
// in parallel.
func TestDirectBytesAllocs(t *testing.T) {
	small, large := appendBytes(1000), appendBytes(4000)
	assert.True(t, large < 6*small, "four times the appends allocate less than six times the bytes")
}

// appendBytes returns the bytes that a check of a history of one committed
// transaction of n appends to one key allocates.
func appendBytes(n int) uint64 {
	ops := make([]any, n)
	for i := range ops {
		ops[i] = appendOf("x", i)
	}
	h := history.New()
	h.Invoke(0, "txn", ops).OK(ops)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	history.Serializable(&matchertest.Seat{}, h, "one transaction")
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}
