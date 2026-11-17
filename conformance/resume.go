// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
)

// The members of a resume vector, of its checks and of its results that the
// path of a fault names, beside the members of a linearizable vector.
const (
	checksMember    = "checks"
	resultsMember   = "results"
	nextCallsMember = "next-calls"
)

// resumeCheck is one check of a resume vector: the number of events of the
// prefix that it reads, and the limits that it states.
type resumeCheck struct {
	Events    *int   `json:"events"`
	Budget    *int   `json:"budget"`
	MemoLimit *int64 `json:"memo-limit"`
}

// resumeResult is the result that a resume vector states for one check: its
// verdict, the detail of its record and the calls of the spec's Next that it
// makes.
type resumeResult struct {
	Expect    string                     `json:"expect"`
	Detail    map[string]json.RawMessage `json:"detail"`
	NextCalls *int                       `json:"next-calls"`
}

// checkResume records the history of a resume vector through one history, up
// to the events of each check in turn. It checks each prefix with
// [history.Linearizable] against the named spec of [Specs] under the limits
// of the check, and every check passes one [history.Checkpoint]. The runner
// counts the calls of the spec's Next that each check makes. It compares the
// verdict, the detail of a failing check and the calls of Next with the
// result that the vector states for the check.
//
// A passing check reports no detail. Each named spec takes one step for each
// call of Next, so the runner compares the steps of a pass with the calls of
// Next that a check of the prefix without the checkpoint makes.
func checkResume(raw json.RawMessage, _ string) error {
	var v struct {
		Spec    string            `json:"spec"`
		History []json.RawMessage `json:"history"`
		Checks  []resumeCheck     `json:"checks"`
		Results []resumeResult    `json:"results"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	spec, named := Specs[v.Spec]
	if !named {
		return fault.At(fault.New("%q is no named spec", v.Spec), fault.Field(specMember))
	}
	if len(v.Results) != len(v.Checks) {
		return fault.At(fault.New("the vector states %d results, want one for each of %d checks",
			len(v.Results), len(v.Checks)), fault.Field(resultsMember))
	}
	calls := 0
	counted := countedSpec(spec, &calls)
	h, opened := history.New(), map[int]history.Call{}
	var cp history.Checkpoint[any]
	read := 0
	for i, c := range v.Checks {
		events, opts, err := c.settings(read, len(v.History))
		if err != nil {
			return fault.At(err, fault.Field(checksMember), fault.Index(i))
		}
		if err := extend(h, opened, v.History[read:events], read); err != nil {
			return err
		}
		read = events
		before := calls
		got := checked(h, counted, append(opts, history.Resume(&cp))...)
		steps := func() int {
			fresh := 0
			checked(h, countedSpec(spec, &fresh), opts...)
			return fresh
		}
		if err := compareResult(got, calls-before, steps, v.Results[i]); err != nil {
			return fault.At(err, fault.Field(resultsMember), fault.Index(i))
		}
	}
	return nil
}

// settings returns the number of events that c reads and the options of its
// limits. read is the number of events that the check before c read, and
// events the number of events of the history. It returns a fault at the
// member of a number of events below 1, below read or past the history, and
// at the member of a limit below 1.
func (c resumeCheck) settings(read, events int) (int, []history.Option, error) {
	least := max(read, 1)
	if c.Events == nil || *c.Events < least || *c.Events > events {
		return 0, nil, fault.At(fault.New("the check reads %s events, want %d to %d", jsonOf(c.Events), least, events),
			fault.Field(eventsMember))
	}
	opts, err := checkOptions(c.Budget, c.MemoLimit, nil)
	return *c.Events, opts, err
}

// countedSpec returns s with a Next that adds one to calls for each call.
func countedSpec(s history.Spec[any], calls *int) history.Spec[any] {
	next := s.Next
	s.Next = func(state any, op history.Operation) []any {
		*calls++
		return next(state, op)
	}
	return s
}

// compareResult returns how c, the call record of a check, differs from the
// result r. calls is the number of calls of the spec's Next that the check
// made. A difference is a verdict other than the expectation of r, a field of
// the detail of a failure that the record states differently, and another
// number of calls of Next. For a pass, it is also steps other than the calls
// of Next of a check of every call, which steps makes.
func compareResult(c call, calls int, steps func() int, r resumeResult) error {
	switch r.Expect {
	case expectFail:
		if err := compareFailure(c, r.Detail); err != nil {
			return err
		}
	case expectPass:
		if c.Verdict != expectPass {
			return fault.At(fault.New("the check ends as %s, want pass", c.Verdict), fault.Field(expectMember))
		}
		path := []fault.Segment{fault.Field(detailMember), fault.Key(stepsMember)}
		var stated int
		if err := json.Unmarshal(r.Detail[stepsMember], &stated); err != nil {
			return fault.At(fault.New("the detail states the steps %s, want a number", r.Detail[stepsMember]), path...)
		}
		if every := steps(); every != stated {
			return fault.At(fault.New("a check of every call takes %d steps, want %d", every, stated), path...)
		}
	default:
		return fault.At(fault.New("the vector expects %q, neither pass nor fail", r.Expect), fault.Field(expectMember))
	}
	if r.NextCalls == nil || *r.NextCalls != calls {
		return fault.At(fault.New("the check calls Next %d times, want %s", calls, jsonOf(r.NextCalls)),
			fault.Field(nextCallsMember))
	}
	return nil
}
