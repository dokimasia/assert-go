// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
)

// violatedDetail is the detail of the record of the check of violatedRead
// against the register, in the history's JSON form, as the reference
// computes it.
const violatedDetail = `{"outcome":"violated","partitions":1,"steps":2,` +
	`"partition":[{"type":"string","value":"x"}],"calls":2,"concurrency":1,` +
	`"linearized":[{"call":0,"completion":1,"process":0,"operation":"write",` +
	`"args":[{"type":"int","value":1}],"output":{"type":"null"}}],` +
	`"states":[{"type":"int","value":1}],` +
	`"candidates":[{"call":2,"completion":3,"process":1,"operation":"read","args":[],` +
	`"output":{"type":"int","value":0}}],"limit":null}`

// TestDetail checks the detail of the record of a failing check: its Go
// values, and its JSON form in the call record of a recorded run.
func TestDetail(t *testing.T) {
	t.Parallel()

	t.Run("Linearizable", func(t *testing.T) {
		t.Parallel()

		t.Run("states the ten fields of the definition with their Go values", func(t *testing.T) {
			t.Parallel()
			want := map[string]any{
				outcomeField:     history.Violated,
				partitionsField:  1,
				stepsField:       2,
				partitionField:   []any{"x"},
				callsField:       2,
				concurrencyField: 1,
				linearizedField: []history.Span{
					{Call: 0, Completion: 1, Op: history.Op{Operation: write, Args: writeOne, Known: true}},
				},
				statesField: []int{1},
				candidatesField: []history.Span{
					{Call: 2, Completion: 3, Process: 1, Op: history.Op{Operation: read, Known: true, Output: 0}},
				},
				limitField: nil,
			}
			assert.Equal(t, detailOf(violatedRead(), register), want, "the detail of the violated read")
		})

		t.Run("states the limit of an undecided check", func(t *testing.T) {
			t.Parallel()
			got := detailOf(violatedRead(), register, history.Budget(1))
			assert.Equal(t, got[limitField], any(history.LimitSteps), "the budget stopped the search")
		})

		t.Run("states the detail in the history's JSON form in the call record", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, recordedDetail(t, violatedRead()), violatedDetail, "the detail of the reference")
		})

		t.Run("states the limit's spelling in the call record of an undecided check", func(t *testing.T) {
			t.Parallel()
			var got struct {
				Limit *string `json:"limit"`
			}
			assert.NoError(t, json.Unmarshal([]byte(recordedDetail(t, violatedRead(), history.Budget(1))), &got),
				"the detail is a JSON object")
			assert.Equal(t, got.Limit, new("steps"), "the spelling of the budget's limit")
		})
	})
}

// recordedDetail checks h against the register on a recorder under opts,
// and returns the detail of the call record of the check as its JSON text.
func recordedDetail(tb testing.TB, h *history.History, opts ...history.Option) string {
	tb.Helper()
	rec := assert.NewRecorder()
	history.Linearizable(rec, h, register, contract, opts...)
	records := rec.Records()
	assert.Length(tb, records, 1, "the call record of the check")
	var record struct {
		Detail json.RawMessage `json:"detail"`
	}
	assert.NoError(tb, json.Unmarshal([]byte(records[0]), &record), "the call record is a JSON object")
	return string(record.Detail)
}
