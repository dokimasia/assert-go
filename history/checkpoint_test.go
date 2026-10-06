// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
)

// TestCheckpoint checks which checks continue the search that a checkpoint
// keeps, through the steps that the spec counts, and that each check
// reports what a search of the whole history reports.
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
				assert.Nil(t, resumedDetail(h, m, &cp), "the write of 1 passes beside a pending write")
				pending.OK(nil)
				recordOK(h, 0, read, nil, 2)
				assert.Nil(t, resumedDetail(h, m, &cp), "the read of 2 follows the write of 2")
				assert.Equal(t, steps, 4, "the first check steps the write of 1, and the second all three calls")
			})

		tests := []struct {
			name  string
			first []history.Option
			again []history.Option
			other bool
		}{
			{
				name:  "searches every call again under another budget",
				first: []history.Option{history.Budget(100)},
				again: []history.Option{history.Budget(99)},
			},
			{
				name:  "searches every call again under another memo limit",
				first: []history.Option{history.MemoLimit(1000)},
				again: []history.Option{history.MemoLimit(999)},
			},
			{
				name:  "searches every call again for another history",
				other: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				steps := 0
				m := counted(&steps)
				h := writes(3)
				var cp history.Checkpoint[int]
				assert.Nil(t, resumedDetail(h, m, &cp, tt.first...), "three writes pass")
				if tt.other {
					h = writes(3)
				}
				recordOK(h, 0, write, writeOne, nil)
				assert.Nil(t, resumedDetail(h, m, &cp, tt.again...), "four writes pass")
				assert.Equal(t, steps, 7, "three steps, and then four for the search of every call")
			})
		}

		t.Run("searches every call again when the memo contains more configurations than the grown search may store",
			func(t *testing.T) {
				t.Parallel()
				h := writes(2)
				var cp history.Checkpoint[int]
				limit := history.MemoLimit(6)
				assert.Nil(t, resumedDetail(h, register, &cp, limit), "two configurations fit a limit of three")
				recordOK(h, 0, write, writeOne, nil)
				recordOK(h, 0, write, writeOne, nil)
				got := resumedDetail(h, register, &cp, limit)
				assert.Equal(t, got, detailOf(h, register, history.Whole(), limit),
					"the search of every call, whose memo stores one configuration of four calls")
				assert.Equal(t, got[limitField], any(history.LimitMemo), "the limit stopped the search")
			})

		t.Run("keeps no search after a check that does not pass", func(t *testing.T) {
			t.Parallel()
			steps := 0
			m := counted(&steps)
			h := violatedRead()
			var cp history.Checkpoint[int]
			first := resumedDetail(h, m, &cp)
			assert.Equal(t, first[outcomeField], any(history.Violated), "the read of 0 follows the write of 1")
			recordOK(h, 2, write, []any{3}, nil)
			assert.Equal(t, resumedDetail(h, m, &cp), detailOf(h, register, history.Whole()),
				"the violation that a search of every call reports")
			assert.Equal(t, steps, 4, "two steps for each search of the write and the read")
		})

		t.Run("keeps no search after a function of the spec panics", func(t *testing.T) {
			t.Parallel()
			steps := 0
			panicking := true
			m := history.Spec[int]{Initial: register.Initial, Next: func(s int, op history.Operation) []int {
				if panicking {
					panic(boom)
				}
				steps++
				return register.Next(s, op)
			}}
			h := writes(2)
			var cp history.Checkpoint[int]
			got := faultOf(t, h, m, history.Whole(), history.Resume(&cp))
			assert.ErrorIs(t, got, history.ErrSpec, "the step of the first write panics")
			panicking = false
			recordOK(h, 0, write, writeOne, nil)
			assert.Nil(t, resumedDetail(h, m, &cp), "three writes pass")
			assert.Equal(t, steps, 3, "the search of every call steps all three")
		})

		t.Run("calls Initial for a history without calls, and continues from its state", func(t *testing.T) {
			t.Parallel()
			initials, steps := 0, 0
			m := counted(&steps)
			m.Initial = func() int {
				initials++
				return 0
			}
			h := history.New()
			var cp history.Checkpoint[int]
			assert.Nil(t, resumedDetail(h, m, &cp), "a history without calls passes")
			recordOK(h, 0, read, nil, 0)
			assert.Nil(t, resumedDetail(h, m, &cp), "the read of the initial state passes")
			assert.Equal(t, []int{initials, steps}, []int{1, 1}, "one Initial, and the step of the read")
		})
	})
}

// resumedDetail checks h against m under Whole, Resume of cp and opts on a
// recorder, and returns the detail of the record of the check, or nil for a
// check that passed.
func resumedDetail(h *history.History, m history.Spec[int], cp *history.Checkpoint[int],
	opts ...history.Option,
) map[string]any {
	return detailOf(h, m, append([]history.Option{history.Whole(), history.Resume(cp)}, opts...)...)
}

// counted returns the register, whose Next adds one to steps for each step.
func counted(steps *int) history.Spec[int] {
	return history.Spec[int]{Initial: register.Initial, Next: func(s int, op history.Operation) []int {
		*steps++
		return register.Next(s, op)
	}}
}

// writes returns a history of n sequential writes of client 0, of the values
// 1 to n.
func writes(n int) *history.History {
	h := history.New()
	for v := range n {
		recordOK(h, 0, write, []any{v + 1}, nil)
	}
	return h
}
