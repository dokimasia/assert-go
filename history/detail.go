// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"encoding/json"
)

// The assertion of a check's record, and the names of its detail fields in
// the order the definition lists them.
const (
	// linearizableID is the assertion of the record.
	linearizableID = "linearizable"
	// outcomeField is how the check ended, a [Verdict].
	outcomeField = "outcome"
	// partitionsField is the number of partitions, an int.
	partitionsField = "partitions"
	// stepsField is the steps of the partitions up to and including the
	// reported one, an int.
	stepsField = "steps"
	// partitionField is the keys of the reported partition, a []any.
	partitionField = "partition"
	// callsField is the number of calls of the reported partition, an int.
	callsField = "calls"
	// concurrencyField is the most calls of the reported partition that are
	// open at one event, an int.
	concurrencyField = "concurrency"
	// linearizedField is the frontier's order of calls, a []Span.
	linearizedField = "linearized"
	// statesField is the frontier's states, a slice of the spec's states.
	statesField = "states"
	// candidatesField is the calls that the spec rejected at the frontier, a
	// []Span.
	candidatesField = "candidates"
	// limitField is the limit that stopped an undecided search, a [Limit].
	limitField = "limit"
)

// detail is the detail of the record of a check: the ten fields of the
// definition. A check that passed states the outcome, the partitions and
// the steps alone.
type detail[S any] struct {
	// outcome is how the check ended.
	outcome Verdict
	// partitions is the number of partitions.
	partitions int
	// steps is the steps of the partitions up to and including the reported
	// one, and of every partition of a check that passed.
	steps int
	// partition are the keys of the reported partition, and none for every
	// key.
	partition []any
	// calls is the number of calls of the reported partition.
	calls int
	// concurrency is the most calls of the reported partition that are open
	// at one event.
	concurrency int
	// linearized is the frontier's order of calls.
	linearized []Span
	// states are the frontier's states.
	states []S
	// candidates are the calls that the spec rejected at the frontier.
	candidates []Span
	// limit is the limit that stopped an undecided search, and zero for a
	// violated one.
	limit Limit
	// final are the states that the order of a passing check leaves, when
	// [Final] asks for them and the check searched one partition or none.
	final []S
}

// reported returns the detail of a check that reports the partition p of
// partitions, whose search ended as e, after steps in its partitions up to
// and including p.
func reported[S any](p partition, e ending[S], partitions, steps int) detail[S] {
	return detail[S]{
		outcome: e.verdict, partitions: partitions, steps: steps, partition: p.keys, calls: len(p.calls),
		concurrency: concurrency(p.calls), linearized: spans(p.calls, e.linearized), states: e.states,
		candidates: spans(p.calls, e.candidates), limit: e.limit,
	}
}

// spans returns the spans of the calls at positions, in order.
func spans(calls []call, positions []int) []Span {
	out := make([]Span, len(positions))
	for i, position := range positions {
		out[i] = calls[position].span
	}
	return out
}

// fields returns the detail as a failure record states it: each field of
// the definition by its name, with its Go value, and nil for the limit of a
// violated check.
func (d detail[S]) fields() map[string]any {
	fields := map[string]any{
		outcomeField:     d.outcome,
		partitionsField:  d.partitions,
		stepsField:       d.steps,
		partitionField:   d.partition,
		callsField:       d.calls,
		concurrencyField: d.concurrency,
		linearizedField:  d.linearized,
		statesField:      d.states,
		candidatesField:  d.candidates,
		limitField:       nil,
	}
	if d.limit != 0 {
		fields[limitField] = d.limit
	}
	return fields
}

// detailJSON is the detail of a failing check as its call record states it,
// in the order that the definition lists its fields.
type detailJSON struct {
	Outcome     Verdict           `json:"outcome"`
	Partitions  int               `json:"partitions"`
	Steps       int               `json:"steps"`
	Partition   []json.RawMessage `json:"partition"`
	Calls       int               `json:"calls"`
	Concurrency int               `json:"concurrency"`
	Linearized  []Span            `json:"linearized"`
	States      []json.RawMessage `json:"states"`
	Candidates  []Span            `json:"candidates"`
	Limit       *Limit            `json:"limit"`
}

// MarshalJSON returns the detail as the call record of a check states it, in
// the form of the definition's vectors: the counts as numbers, each key and
// each state as a typed literal, each call in the history's JSON form, and
// null for the limit of a violated check.
func (d detail[S]) MarshalJSON() ([]byte, error) {
	out := detailJSON{
		Outcome: d.outcome, Partitions: d.partitions, Steps: d.steps, Partition: literals(d.partition),
		Calls: d.calls, Concurrency: d.concurrency, Linearized: d.linearized, States: literals(d.states),
		Candidates: d.candidates,
	}
	if d.limit != 0 {
		out.Limit = &d.limit
	}
	return json.Marshal(out)
}
