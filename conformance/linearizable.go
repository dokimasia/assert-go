// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"maps"
	"slices"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
)

// linearizableContract is the contract of the check of a linearizable
// vector.
const linearizableContract = "the history of the vector is linearizable"

// The members of a linearizable vector and of its detail that the path of a
// fault names, beside the members of every vector.
const (
	specMember       = "spec"
	historyMember    = "history"
	budgetMember     = "budget"
	memoLimitMember  = "memo-limit"
	workersMember    = "workers"
	stepsMember      = "steps"
	partitionsMember = "partitions"
)

// undecidedDetail is what the runner reads of the detail of a check that
// passes the budget of a passing vector less one step.
type undecidedDetail struct {
	Outcome    string  `json:"outcome"`
	Partitions int     `json:"partitions"`
	Steps      int     `json:"steps"`
	Limit      *string `json:"limit"`
}

// checkLinearizable records the history of a linearizable vector, checks it
// with [history.Linearizable] against the named spec of [Specs] on a
// recorder, under the vector's options, and compares the outcome with the
// one that the vector states. A failing vector compares each field of the
// detail of the call record with the vector's.
//
// A passing check reports no detail, so the runner observes the steps of a
// passing vector through two more checks: under a budget of the steps it
// passes, and under a budget of one step less it ends undecided at the steps
// limit after that budget, in the vector's partitions. The budget applies to
// each partition, so this fixes the steps of a pass over one partition.
func checkLinearizable(raw json.RawMessage, _ string) error {
	var v struct {
		Spec      string                     `json:"spec"`
		History   []json.RawMessage          `json:"history"`
		Budget    *int                       `json:"budget"`
		MemoLimit *int64                     `json:"memo-limit"`
		Workers   *int                       `json:"workers"`
		Expect    string                     `json:"expect"`
		Detail    map[string]json.RawMessage `json:"detail"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	spec, named := Specs[v.Spec]
	if !named {
		return fault.At(fault.New("%q is no named spec", v.Spec), fault.Field(specMember))
	}
	h, err := historyOf(v.History)
	if err != nil {
		return err
	}
	opts, err := checkOptions(v.Budget, v.MemoLimit, v.Workers)
	if err != nil {
		return err
	}
	c := checked(h, spec, opts...)
	switch v.Expect {
	case expectFail:
		return compareFailure(c, v.Detail)
	case expectPass:
		return comparePass(c, v.Detail, func(budget int) call {
			return checked(h, spec, append(slices.Clip(opts), history.Budget(budget))...)
		})
	}
	return fault.At(fault.New("the vector expects %q, neither pass nor fail", v.Expect), fault.Field(expectMember))
}

// checkOptions returns the options of a check that a vector states: its
// budget, its memo limit and its workers, each when it states one. It
// returns a fault at the member of a value below 1.
func checkOptions(budget *int, memoLimit *int64, workers *int) ([]history.Option, error) {
	var opts []history.Option
	if budget != nil {
		if *budget < 1 {
			return nil, fault.At(fault.New("the budget is %d, below 1", *budget), fault.Field(budgetMember))
		}
		opts = append(opts, history.Budget(*budget))
	}
	if memoLimit != nil {
		if *memoLimit < 1 {
			return nil, fault.At(fault.New("the memo limit is %d, below 1", *memoLimit), fault.Field(memoLimitMember))
		}
		opts = append(opts, history.MemoLimit(*memoLimit))
	}
	if workers != nil {
		if *workers < 1 {
			return nil, fault.At(fault.New("the workers are %d, below 1", *workers), fault.Field(workersMember))
		}
		opts = append(opts, history.Workers(*workers))
	}
	return opts, nil
}

// checked checks h against s under opts on a recorder, and returns the call
// record of the check.
func checked(h *history.History, s history.Spec[any], opts ...history.Option) call {
	rec := assert.NewRecorder()
	history.Linearizable(rec, h, s, linearizableContract, opts...)
	// A recorder keeps the call record of every call.
	return callsOf(rec)[0]
}

// compareFailure returns how c, the call record of a check, differs from a
// failure with the detail stated: a verdict other than fail, or a field of
// the detail that the record states differently, in the order of the
// fields' names.
func compareFailure(c call, stated map[string]json.RawMessage) error {
	if c.Verdict != expectFail {
		return fault.At(fault.New("the check ends as %s, want fail", c.Verdict), fault.Field(expectMember))
	}
	var detail map[string]json.RawMessage
	// The detail of a failing check is a JSON object.
	_ = json.Unmarshal(c.Detail, &detail)
	for _, name := range slices.Sorted(maps.Keys(stated)) {
		path := []fault.Segment{fault.Field(detailMember), fault.Key(name)}
		got, present := detail[name]
		if !present {
			return fault.At(fault.New("the record states no such field, want %s", stated[name]), path...)
		}
		if !sameJSON(got, stated[name]) {
			return fault.At(fault.New("the field is %s, want %s", got, stated[name]), path...)
		}
	}
	return nil
}

// comparePass returns how c, the call record of a check, differs from a pass
// whose detail states its steps and its partitions: a verdict other than
// pass, a check under the budget of the steps, run by within, that does not
// pass, or one under a budget of one step less that ends other than
// undecided at the steps limit after that budget in the stated partitions.
// It returns a fault for a pass over other than one partition or in fewer
// than two steps, whose steps no record states.
func comparePass(c call, stated map[string]json.RawMessage, within func(budget int) call) error {
	if c.Verdict != expectPass {
		return fault.At(fault.New("the check ends as %s, want pass", c.Verdict), fault.Field(expectMember))
	}
	var steps, partitions int
	stepsErr := json.Unmarshal(stated[stepsMember], &steps)
	partitionsErr := json.Unmarshal(stated[partitionsMember], &partitions)
	if stepsErr != nil || partitionsErr != nil || partitions != 1 || steps < 2 {
		return fault.At(fault.New(
			"the runner observes the steps of a pass over one partition in two steps or more, "+
				"and the vector states %s partitions and %s steps", stated[partitionsMember], stated[stepsMember]),
			fault.Field(detailMember))
	}
	path := []fault.Segment{fault.Field(detailMember), fault.Key(stepsMember)}
	if exact := within(steps); exact.Verdict != expectPass {
		return fault.At(fault.New("the check ends as %s within %d steps, want pass", exact.Verdict, steps), path...)
	}
	under := within(steps - 1)
	var d undecidedDetail
	// The detail of a failing check is a JSON object, and a pass states none.
	_ = json.Unmarshal(under.Detail, &d)
	limit := history.LimitSteps.String()
	want := undecidedDetail{
		Outcome:    history.Undecided.String(),
		Partitions: partitions,
		Steps:      steps - 1,
		Limit:      &limit,
	}
	if under.Verdict != expectFail || d.Limit == nil || *d.Limit != limit ||
		d.Outcome != want.Outcome || d.Partitions != want.Partitions || d.Steps != want.Steps {

		return fault.At(fault.New("the check within %d steps ends as %s with %s, want %s",
			steps-1, under.Verdict, jsonOf(d), jsonOf(want)), path...)
	}
	return nil
}
