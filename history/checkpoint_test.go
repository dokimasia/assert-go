// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/prop"
)

// partitionKeys are the keys that the writes of a growing history of two
// partitions take in turn.
var partitionKeys = []any{"x", "y"}

// TestCheckpoint checks which checks continue the searches that a checkpoint
// keeps, through the steps that the spec counts, and that each check
// reports what a check of every call reports.
func TestCheckpoint(t *testing.T) {
	t.Parallel()

	t.Run("Linearizable", func(t *testing.T) {
		t.Parallel()

		t.Run("continues the kept search, so 20 checks after 20 calls take 20 steps", func(t *testing.T) {
			t.Parallel()
			steps := 0
			m := counted(&steps)
			h := history.New()
			var cp history.Checkpoint[int]
			for v := range 20 {
				recordOK(h, 0, write, []any{v}, nil)
				assert.Nil(t, resumedDetail(h, m, &cp, history.Whole()), "the history of writes is linearizable")
			}
			assert.Equal(t, steps, 20, "one step for each call")
		})

		t.Run("continues the search of each partition, so 20 checks after 20 calls over two keys take 20 steps",
			func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := history.New()
				var cp history.Checkpoint[int]
				for v := range 20 {
					recordOK(h, 0, write, []any{v}, nil, partitionKeys[v%len(partitionKeys)])
					assert.Nil(t, resumedDetail(h, m, &cp), "the history of writes is linearizable")
				}
				assert.Equal(t, steps, 20, "one step for each call")
			})

		t.Run("searches every call again after the history completes a call that the kept search read as pending",
			func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := history.New()
				var cp history.Checkpoint[int]
				recordOK(h, 0, write, writeOne, nil)
				pending := h.Invoke(1, write, []any{2})
				assert.Nil(t, resumedDetail(h, m, &cp, history.Whole()), "the write of 1 passes beside a pending write")
				pending.OK(nil)
				recordOK(h, 0, read, nil, 2)
				assert.Nil(t, resumedDetail(h, m, &cp, history.Whole()), "the read of 2 follows the write of 2")
				assert.Equal(t, steps, 4, "the first check steps the write of 1, and the second all three calls")
			})

		t.Run("searches the partition of a call that the kept check read as pending again after the call completes",
			func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := history.New()
				var cp history.Checkpoint[int]
				recordOK(h, 0, write, writeOne, nil, "x")
				recordOK(h, 0, write, []any{2}, nil, "y")
				pending := h.Invoke(1, write, []any{3}, "y")
				assert.Nil(t, resumedDetail(h, m, &cp), "the writes of 1 and 2 pass beside a pending write")
				pending.OK(nil)
				recordOK(h, 0, read, nil, 3, "y")
				assert.Nil(t, resumedDetail(h, m, &cp), "the read of 3 follows the write of 3")
				assert.Equal(t, steps, 5, "two steps, and then the three calls of y from its first call")
			})

		t.Run("searches the partition that a new call joins to another again, and continues the others",
			func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := history.New()
				var cp history.Checkpoint[int]
				recordOK(h, 0, write, writeOne, nil, "x")
				recordOK(h, 0, write, []any{2}, nil, "y")
				recordOK(h, 0, write, []any{3}, nil, "z")
				assert.Nil(t, resumedDetail(h, m, &cp), "the three writes pass apart")
				recordOK(h, 0, read, nil, 2, "x", "y", "w")
				assert.Nil(t, resumedDetail(h, m, &cp), "the read of 2 follows the writes of 1 and 2")
				assert.Equal(t, steps, 6, "three steps, and then the three calls of x and y from their first call")
			})

		t.Run("joins each partition of the keys of a call once", func(t *testing.T) {
			t.Parallel()
			steps := 0
			m := counted(&steps)
			h := history.New()
			var cp history.Checkpoint[int]
			recordOK(h, 0, write, writeOne, nil, "x", "z")
			recordOK(h, 0, write, []any{2}, nil, "y")
			assert.Nil(t, resumedDetail(h, m, &cp), "the two writes pass apart")
			recordOK(h, 0, read, nil, 2, "x", "z", "y")
			assert.Nil(t, resumedDetail(h, m, &cp), "the read of 2 follows the writes of 1 and 2")
			assert.Equal(t, steps, 5, "two steps, and then the three calls of x, y and z from their first call")
		})

		t.Run("continues the search of a partition that a call joins with a key of no partition", func(t *testing.T) {
			t.Parallel()
			steps := 0
			m := counted(&steps)
			h := history.New()
			var cp history.Checkpoint[int]
			recordOK(h, 0, write, writeOne, nil, "x", "z")
			assert.Nil(t, resumedDetail(h, m, &cp), "the write passes")
			recordOK(h, 0, read, nil, 1, "x", "z", "y")
			assert.Nil(t, resumedDetail(h, m, &cp), "the read of 1 follows the write")
			assert.Equal(t, steps, 2, "one step for each call")
		})

		t.Run("searches every partition as one after a call without keys, which every later call joins",
			func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := history.New()
				var cp history.Checkpoint[int]
				recordOK(h, 0, write, writeOne, nil, "x")
				recordOK(h, 0, write, []any{2}, nil, "y")
				assert.Nil(t, resumedDetail(h, m, &cp), "the two writes pass apart")
				recordOK(h, 0, read, nil, 2)
				assert.Nil(t, resumedDetail(h, m, &cp), "the read of 2 follows both writes")
				recordOK(h, 0, read, nil, 2, "z")
				assert.Nil(t, resumedDetail(h, m, &cp), "the read of 2 on z follows the read of every key")
				assert.Equal(t, steps, 6, "two steps, the three calls from the first call, and the read on z")
			})

		t.Run("continues the search of the one partition that a call without keys joins", func(t *testing.T) {
			t.Parallel()
			steps := 0
			m := counted(&steps)
			h := history.New()
			var cp history.Checkpoint[int]
			recordOK(h, 0, write, writeOne, nil, "x")
			assert.Nil(t, resumedDetail(h, m, &cp), "the write passes")
			recordOK(h, 0, read, nil, 1)
			assert.Nil(t, resumedDetail(h, m, &cp), "the read of 1 follows the write")
			assert.Equal(t, steps, 2, "one step for each call")
		})

		t.Run("divides the partition of a pending call that joined two keys once the call fails", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			var cp history.Checkpoint[int]
			recordOK(h, 0, write, writeOne, nil, "x")
			recordOK(h, 0, write, []any{2}, nil, "y")
			pending := h.Invoke(1, write, []any{3}, "x", "y")
			assert.Nil(t, resumedDetail(h, register, &cp), "the writes of 1 and 2 pass beside a pending write")
			pending.Fail(errRefused)
			recordOK(h, 0, read, nil, 2, "x")
			got := resumedDetail(h, register, &cp)
			assert.Equal(t, got, detailOf(h, register), "the record of a check of every call")
			assert.Equal(t, []any{got[partitionsField], got[partitionField]}, []any{2, []any{"x"}},
				"the read of 2 on x violates the partition of x alone")
		})

		tests := []struct {
			name  string
			first []history.Option
			again []history.Option
			other bool
		}{
			{
				name:  "searches every call again under another budget",
				first: []history.Option{history.Whole(), history.Budget(100)},
				again: []history.Option{history.Whole(), history.Budget(99)},
			},
			{
				name:  "searches every call again under another memo limit",
				first: []history.Option{history.Whole(), history.MemoLimit(1000)},
				again: []history.Option{history.Whole(), history.MemoLimit(999)},
			},
			{
				name:  "searches every call again for another history",
				first: []history.Option{history.Whole()},
				again: []history.Option{history.Whole()},
				other: true,
			},
			{
				name:  "searches every partition again under another budget",
				first: []history.Option{history.Budget(100)},
				again: []history.Option{history.Budget(99)},
			},
			{
				name:  "searches every partition again under another memo limit",
				first: []history.Option{history.MemoLimit(1000)},
				again: []history.Option{history.MemoLimit(999)},
			},
			{
				name:  "searches every partition again for another history",
				other: true,
			},
			{
				name:  "searches every partition again after a check of the whole history",
				first: []history.Option{history.Whole()},
			},
			{
				name:  "searches every call again after a check of each partition",
				again: []history.Option{history.Whole()},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := twoKeys()
				var cp history.Checkpoint[int]
				assert.Nil(t, resumedDetail(h, m, &cp, tt.first...), "the writes of 1 and 2 pass")
				if tt.other {
					h = twoKeys()
				}
				recordOK(h, 0, write, []any{3}, nil, "x")
				assert.Nil(t, resumedDetail(h, m, &cp, tt.again...), "the three writes pass")
				assert.Equal(t, steps, 5, "two steps, and then three for the search of every call")
			})
		}

		t.Run("searches every call again when the memo contains more configurations than the grown search may store",
			func(t *testing.T) {
				t.Parallel()
				h := writes(2)
				var cp history.Checkpoint[int]
				limit := history.MemoLimit(6)
				assert.Nil(t, resumedDetail(h, register, &cp, history.Whole(), limit),
					"two configurations fit a limit of three")
				recordOK(h, 0, write, writeOne, nil)
				recordOK(h, 0, write, writeOne, nil)
				got := resumedDetail(h, register, &cp, history.Whole(), limit)
				assert.Equal(t, got, detailOf(h, register, history.Whole(), limit),
					"the search of every call, whose memo stores one configuration of four calls")
				assert.Equal(t, got[limitField], any(history.LimitMemo), "the limit stopped the search")
			})

		t.Run("searches a partition again when its memo contains more configurations than its grown search may store",
			func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := history.New()
				var cp history.Checkpoint[int]
				limit := history.MemoLimit(6)
				recordOK(h, 0, write, writeOne, nil, "y")
				recordOK(h, 0, write, writeOne, nil, "x")
				recordOK(h, 0, write, []any{2}, nil, "x")
				assert.Nil(t, resumedDetail(h, m, &cp, limit), "the configurations of x and of y fit a limit of six")
				recordOK(h, 0, write, []any{3}, nil, "x")
				recordOK(h, 0, write, []any{4}, nil, "x")
				got := resumedDetail(h, m, &cp, limit)
				assert.Equal(t, got, detailOf(h, register, limit),
					"the record of a check of every call, whose memo of x stores one configuration of four calls")
				assert.Equal(t, []any{got[limitField], steps}, []any{history.LimitMemo, 5},
					"three steps, and then the two steps of x from its first call")
			})

		t.Run("continues the search of a partition whose memo fits the capacity of its grown search",
			func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := history.New()
				var cp history.Checkpoint[int]
				limit := history.MemoLimit(9)
				recordOK(h, 0, write, writeOne, nil, "x")
				recordOK(h, 0, write, []any{2}, nil, "x")
				assert.Nil(t, resumedDetail(h, m, &cp, limit), "two configurations of two calls fit nine bits")
				recordOK(h, 0, write, []any{3}, nil, "x")
				assert.Nil(t, resumedDetail(h, m, &cp, limit), "three configurations of three calls fit nine bits")
				assert.Equal(t, steps, 3, "one step for each call")
			})

		t.Run("keeps no search after a check that does not pass", func(t *testing.T) {
			t.Parallel()
			steps := 0
			m := counted(&steps)
			h := violatedRead()
			var cp history.Checkpoint[int]
			first := resumedDetail(h, m, &cp, history.Whole())
			assert.Equal(t, first[outcomeField], any(history.Violated), "the read of 0 follows the write of 1")
			recordOK(h, 2, write, []any{3}, nil)
			assert.Equal(t, resumedDetail(h, m, &cp, history.Whole()), detailOf(h, register, history.Whole()),
				"the violation that a search of every call reports")
			assert.Equal(t, steps, 4, "two steps for each search of the write and the read")
		})

		t.Run("keeps no search after a check of each partition that does not pass", func(t *testing.T) {
			t.Parallel()
			steps := 0
			m := counted(&steps)
			h := violatedRead()
			recordOK(h, 2, write, []any{2}, nil, "y")
			var cp history.Checkpoint[int]
			first := resumedDetail(h, m, &cp)
			assert.Equal(t, first[outcomeField], any(history.Violated), "the read of 0 follows the write of 1")
			recordOK(h, 2, write, []any{3}, nil, "y")
			assert.Equal(t, resumedDetail(h, m, &cp), detailOf(h, register),
				"the violation that a check of every call reports")
			assert.Equal(t, steps, 4, "two steps for each search of the partition of x")
		})

		t.Run("keeps no search after a function of the spec panics", func(t *testing.T) {
			t.Parallel()
			panicking := true
			m := panicsUntil(&panicking)
			h := writes(2)
			var cp history.Checkpoint[int]
			got := faultOf(t, h, m, history.Whole(), history.Resume(&cp))
			assert.ErrorIs(t, got, history.ErrSpec, "the step of the first write panics")
			panicking = false
			recordOK(h, 0, read, nil, 1)
			assert.Equal(t, resumedDetail(h, m, &cp, history.Whole()), detailOf(h, register, history.Whole()),
				"the violation that a search of every call reports")
		})

		t.Run("keeps no search after a function of the spec panics in a check of each partition", func(t *testing.T) {
			t.Parallel()
			panicking := true
			m := panicsUntil(&panicking)
			h := twoKeys()
			var cp history.Checkpoint[int]
			got := faultOf(t, h, m, history.Resume(&cp))
			assert.ErrorIs(t, got, history.ErrSpec, "the step of the write of x panics")
			panicking = false
			recordOK(h, 0, read, nil, 2, "x")
			assert.Equal(t, resumedDetail(h, m, &cp), detailOf(h, register),
				"the violation that a check of every call reports")
		})

		t.Run("calls Initial for a history without calls, and continues from its state", func(t *testing.T) {
			t.Parallel()
			initials, steps := 0, 0
			m := counted(&steps)
			m.Initial = initialOf(&initials)
			h := history.New()
			var cp history.Checkpoint[int]
			assert.Nil(t, resumedDetail(h, m, &cp, history.Whole()), "a history without calls passes")
			recordOK(h, 0, read, nil, 0)
			assert.Nil(t, resumedDetail(h, m, &cp, history.Whole()), "the read of the initial state passes")
			assert.Equal(t, []int{initials, steps}, []int{1, 1}, "one Initial, and the step of the read")
		})

		t.Run("calls no function of the spec for a history without calls in a check of each partition",
			func(t *testing.T) {
				t.Parallel()
				initials, steps := 0, 0
				m := counted(&steps)
				m.Initial = initialOf(&initials)
				h := history.New()
				var cp history.Checkpoint[int]
				assert.Nil(t, resumedDetail(h, m, &cp), "a history without calls passes")
				assert.Equal(t, initials, 0, "no partition calls Initial")
				recordOK(h, 0, read, nil, 0, "x")
				assert.Nil(t, resumedDetail(h, m, &cp), "the read of the initial state passes")
				assert.Equal(t, []int{initials, steps}, []int{1, 1}, "the Initial of x, and the step of the read")
			})

		t.Run("reports at each check of a growing history what a check of every call reports", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, "a check that continues each partition reports what a check of every call reports",
				func(c *prop.Case) {
					m := register
					if c.Draw(prop.Boolean(), "lossy") {
						m = lossy
					}
					workers := history.Workers(c.Draw(prop.Integer(1, 2), "workers"))
					h := history.New()
					var cp history.Checkpoint[int]
					open := map[int]history.Call{}
					for range c.Draw(prop.Integer(1, 30), "events") {
						client := c.Draw(prop.Integer(0, 2), "client")
						if call, busy := open[client]; busy {
							delete(open, client)
							complete(c, call)
						} else {
							open[client] = invoke(c, h, client)
						}
						got, gotFinal := outcomeOf(h, m, workers, history.Resume(&cp))
						want, wantFinal := outcomeOf(h, m, workers)
						assert.Equal(c, got, want, "the check reports the records of a check of every call")
						assert.Equal(c, gotFinal, wantFinal, "the check leaves the same states")
					}
				}, prop.Seed(1), prop.Cases(300))
		})
	})
}

// twoKeys returns a history in which client 0 writes 1 on x and then 2 on
// y.
func twoKeys() *history.History {
	h := history.New()
	recordOK(h, 0, write, writeOne, nil, "x")
	recordOK(h, 0, write, []any{2}, nil, "y")
	return h
}

// panicsUntil returns the register, whose Next panics while panicking is
// set.
func panicsUntil(panicking *bool) history.Spec[int] {
	return history.Spec[int]{Initial: register.Initial, Next: func(s int, op history.Operation) []int {
		if *panicking {
			panic(boom)
		}
		return register.Next(s, op)
	}}
}
