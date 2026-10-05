// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"errors"
	"slices"

	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
)

// The members of a seam vector, of a script entry and of an interval entry
// that the path of a fault names, beside the members of every vector.
const (
	scriptMember    = "script"
	intervalsMember = "intervals"
	eventsMember    = "events"
	refusedMember   = "refused"
	outputMember    = "output"
)

// completions maps the completion kind that an interval entry states to its
// kind of event.
var completions = map[string]history.Kind{"ok": history.OK, "fail": history.Fail, "unknown": history.Unknown}

// scriptEntry is one entry of a script, the form in which a vector states a
// history. An entry with invoke opens a call under a number of the script,
// and an entry with ok, fail or unknown completes the call it names.
type scriptEntry struct {
	Invoke    *int              `json:"invoke"`
	OK        *int              `json:"ok"`
	Fail      *int              `json:"fail"`
	Unknown   *int              `json:"unknown"`
	Client    int               `json:"client"`
	Operation string            `json:"operation"`
	Args      []json.RawMessage `json:"args"`
	Keys      []json.RawMessage `json:"keys"`
	Output    json.RawMessage   `json:"output"`
	Error     string            `json:"error"`
}

// intervalEntry is one entry of the intervals of a seam vector: a call with
// a start and an end on one clock. An entry states a completion kind and an
// end, or neither for a pending call.
type intervalEntry struct {
	Client    int               `json:"client"`
	Operation string            `json:"operation"`
	Args      []json.RawMessage `json:"args"`
	Keys      []json.RawMessage `json:"keys"`
	Start     int64             `json:"start"`
	End       *int64            `json:"end"`
	Kind      string            `json:"kind"`
	Output    json.RawMessage   `json:"output"`
	Error     string            `json:"error"`
}

// checkSeam records the calls of a seam vector, a script through
// [history.New] or intervals through [history.FromIntervals], and compares
// the events in the history's JSON form, or the entry that the seam refuses,
// with the ones that the vector states.
func checkSeam(raw json.RawMessage, _ string) error {
	var v struct {
		Script    []json.RawMessage `json:"script"`
		Intervals []intervalEntry   `json:"intervals"`
		Events    []json.RawMessage `json:"events"`
		Refused   *int              `json:"refused"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	if (v.Script == nil) == (v.Intervals == nil) {
		return fault.New("the vector states a script or intervals")
	}
	var h *history.History
	var err error
	if v.Script != nil {
		h, err = fromScript(v.Script, v.Refused)
	} else {
		h, err = fromIntervals(v.Intervals, v.Refused)
	}
	if err != nil || v.Refused != nil {
		return err
	}
	events := h.Events()
	for i, e := range events[:min(len(events), len(v.Events))] {
		// An event of a history that a vector records marshals.
		data, _ := e.MarshalJSON()
		if !sameJSON(data, v.Events[i]) {
			return fault.At(fault.New("the event is %s, want %s", data, v.Events[i]),
				fault.Field(eventsMember), fault.Index(i))
		}
	}
	if len(events) != len(v.Events) {
		return fault.At(fault.New("the history records %d events, want %d", len(events), len(v.Events)),
			fault.Field(eventsMember))
	}
	return nil
}

// fromScript records script through a history, and checks that the seam
// refuses the entry refused, or none for a nil refused. It returns a fault
// at the script for a malformed entry, and at the member refused for a
// refusal that differs.
func fromScript(script []json.RawMessage, refused *int) (*history.History, error) {
	h, entry, err := record(script)
	if err != nil {
		return nil, fault.At(err, fault.Field(scriptMember))
	}
	want := -1
	if refused != nil {
		want = *refused
	}
	if entry == want {
		return h, nil
	}
	if entry == -1 {
		return nil, fault.At(fault.New("the seam refuses no entry, want entry %d", want), fault.Field(refusedMember))
	}
	return nil, fault.At(fault.New("the seam refuses entry %d, want %s", entry, jsonOf(refused)),
		fault.Field(refusedMember))
}

// historyOf records script, the history of a vector that a check reads,
// through a history. It returns a fault at the member history for a
// malformed entry, and for an entry that the seam refuses.
func historyOf(script []json.RawMessage) (*history.History, error) {
	h, refused, err := record(script)
	if err != nil {
		return nil, fault.At(err, fault.Field(historyMember))
	}
	if refused != -1 {
		return nil, fault.At(fault.New("the seam refuses the entry"), fault.Field(historyMember), fault.Index(refused))
	}
	return h, nil
}

// record records script through a history, and returns the history and the
// entry whose call of the seam panics as a usage error, or -1 when the seam
// refuses none. It records no entry after the refused one. It returns a
// fault at the entry for an entry that states no call of the seam, a value
// that is no typed literal, and a completion of a call that the script did
// not open.
func record(script []json.RawMessage) (*history.History, int, error) {
	h := history.New()
	calls := map[int]history.Call{}
	for i, raw := range script {
		var e scriptEntry
		if err := decode(raw, &e); err != nil {
			return nil, 0, fault.At(err, fault.Index(i))
		}
		apply, err := e.call(h, calls)
		if err != nil {
			return nil, 0, fault.At(err, fault.Index(i))
		}
		if refuses(apply) {
			return h, i, nil
		}
	}
	return h, -1, nil
}

// call returns the call of the seam that e states, on h and on the calls
// that the script opened, by their numbers. It returns a fault for an entry
// that states no call of the seam, a value that is no typed literal, and a
// completion of a call that the script did not open.
func (e scriptEntry) call(h *history.History, calls map[int]history.Call) (func(), error) {
	if e.Invoke != nil {
		args, err := decodeValues(e.Args)
		if err != nil {
			return nil, fault.At(err, fault.Field(argsMember))
		}
		keys, err := decodeValues(e.Keys)
		if err != nil {
			return nil, fault.At(err, fault.Field(keysMember))
		}
		return func() { calls[*e.Invoke] = h.Invoke(e.Client, e.Operation, args, keys...) }, nil
	}
	if e.OK != nil {
		output, err := literal.Decode(e.Output)
		if err != nil {
			return nil, fault.At(err, fault.Field(outputMember))
		}
		return completing(calls, *e.OK, func(c history.Call) { c.OK(output) })
	}
	if e.Fail != nil {
		return completing(calls, *e.Fail, func(c history.Call) { c.Fail(errors.New(e.Error)) })
	}
	if e.Unknown != nil {
		return completing(calls, *e.Unknown, func(c history.Call) { c.Unknown(errors.New(e.Error)) })
	}
	return nil, fault.New("the entry states no call of the seam")
}

// completing returns complete applied to the call number of calls, and a
// fault for a call that the script did not open.
func completing(calls map[int]history.Call, number int, complete func(history.Call)) (func(), error) {
	c, opened := calls[number]
	if !opened {
		return nil, fault.New("the entry completes call %d, which the script does not open", number)
	}
	return func() { complete(c) }, nil
}

// refuses runs apply, a call of the seam, and reports whether it panics as
// a usage error.
func refuses(apply func()) (refused bool) {
	defer func() { refused = recover() != nil }()
	apply()
	return false
}

// fromIntervals builds the history of entries with history.FromIntervals,
// and checks that it refuses the entry refused, or none for a nil refused.
// It returns a fault at the intervals for a malformed entry, and at the
// member refused for a refusal that differs.
func fromIntervals(entries []intervalEntry, refused *int) (*history.History, error) {
	intervals := make([]history.Interval, len(entries))
	for i, e := range entries {
		interval, err := e.interval()
		if err != nil {
			return nil, fault.At(err, fault.Field(intervalsMember), fault.Index(i))
		}
		intervals[i] = interval
	}
	h, err := history.FromIntervals(intervals)
	if refused == nil && err == nil {
		return h, nil
	}
	var f *fault.Error
	if refused != nil && errors.Is(err, history.ErrInterval) && errors.As(err, &f) &&
		slices.Equal(f.Path, fault.Path{fault.Index(*refused)}) {
		return nil, nil
	}
	return nil, fault.At(fault.New("from-intervals returns %v, want a refusal of %s", err, jsonOf(refused)),
		fault.Field(refusedMember))
}

// interval returns the interval that e states. It returns a fault for an
// entry that states an end without a completion kind or the reverse, a kind
// that completes no call, and a value that is no typed literal.
func (e intervalEntry) interval() (history.Interval, error) {
	args, err := decodeValues(e.Args)
	if err != nil {
		return history.Interval{}, fault.At(err, fault.Field(argsMember))
	}
	keys, err := decodeValues(e.Keys)
	if err != nil {
		return history.Interval{}, fault.At(err, fault.Field(keysMember))
	}
	i := history.Interval{Client: e.Client, Operation: e.Operation, Args: args, Keys: keys, Start: e.Start}
	if e.End == nil && e.Kind == "" {
		i.Kind = history.Invoke
		return i, nil
	}
	kind, completes := completions[e.Kind]
	if e.End == nil || !completes {
		return history.Interval{}, fault.New("the entry states a completion kind and an end, or neither")
	}
	i.End, i.Kind = *e.End, kind
	if kind != history.OK {
		i.Error = errors.New(e.Error)
		return i, nil
	}
	if i.Output, err = literal.Decode(e.Output); err != nil {
		return history.Interval{}, fault.At(err, fault.Field(outputMember))
	}
	return i, nil
}
