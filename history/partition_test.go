// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
)

// TestPartition checks how a check reads the calls of a history and divides
// them into partitions, through the record of the check.
func TestPartition(t *testing.T) {
	t.Parallel()

	t.Run("Linearizable", func(t *testing.T) {
		t.Parallel()

		t.Run("removes the calls that failed", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, write, writeOne, "x").Fail(errRefused)
			recordOK(h, 0, write, []any{2}, nil, "x")
			recordOK(h, 1, read, nil, 1, "x")
			got := detailOf(h, register)
			assert.Equal(t, []any{got[outcomeField], got[callsField]}, []any{history.Violated, 2},
				"the failed write took no effect, and the partition lists the other two calls")
		})

		t.Run("passes a history without calls", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, detailOf(history.New(), register), "no partition")
		})

		t.Run("passes a history whose calls all failed", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, read, nil, "x").Fail(errRefused)
			assert.Nil(t, detailOf(h, register), "no call is left")
		})

		t.Run("searches the partitions in the order of their first invocation", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil, "b")
			recordOK(h, 0, write, writeOne, nil, "a")
			recordOK(h, 1, read, nil, 1, "b")
			recordOK(h, 1, read, nil, 0, "a")
			got := detailOf(h, register)
			assert.Equal(t, []any{got[partitionsField], got[partitionField], got[stepsField]},
				[]any{2, []any{"a"}, 4}, "the passing partition of b first, then the violated one of a")
		})

		t.Run("joins two calls that share a key, and lists the keys once in the order first declared",
			func(t *testing.T) {
				t.Parallel()
				h := history.New()
				recordOK(h, 0, write, writeOne, nil, "y")
				recordOK(h, 0, write, []any{2}, nil, "x")
				recordOK(h, 1, read, nil, 0, "x", "y", "x")
				got := detailOf(h, register)
				assert.Equal(t, []any{got[partitionsField], got[partitionField], got[callsField]},
					[]any{1, []any{"y", "x"}, 3}, "the read on x and y joins the writes")
			})

		t.Run("puts every call into one partition of every key for a call without keys", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, write, writeOne, nil, "x")
			recordOK(h, 1, read, nil, 0)
			got := detailOf(h, register)
			assert.Equal(t, []any{got[partitionsField], got[partitionField], got[callsField]},
				[]any{1, []any{}, 2}, "an empty list of keys")
		})

		t.Run("states the most calls open at one event, a call that is not known open to the end", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			lost := h.Invoke(0, write, writeOne, "x")
			two := h.Invoke(1, write, []any{2}, "x")
			lost.Unknown(errRefused)
			seven := h.Invoke(2, read, nil, "x")
			two.OK(nil)
			seven.OK(7)
			h.Invoke(3, write, []any{3}, "x")
			got := detailOf(h, register)
			assert.Equal(t, []any{got[callsField], got[concurrencyField], got[stepsField]}, []any{4, 3, 9},
				"the unknown write, the second write and the read are open at the read's invocation")
		})
	})
}
