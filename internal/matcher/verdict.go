// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/record"
)

// Pass reports that a call of assertion passed. It writes the call's record
// when its seat records calls, and reports nothing to the seat.
//
// A pass sends the seat nothing, so it marks no frame as a helper of the
// seat. On a switch of recording that states no value, Pass and the frames
// under it mark themselves before the seat receives the switch's fault, so
// the fault names the caller's line.
//
// # Allocation contract
//
// Pass allocates nothing for a seat whose calls are not recorded, such as
// a test's seat with recording off.
func Pass(seat Seat, mode Mode, assertion, contract string) {
	if _, err := record.On(); err != nil {
		seat.Helper()
	}
	Running{seat: seat}.Pass(mode, assertion, contract)
}

// Fail reports that a call of assertion failed with detail, which contains
// exactly the fields that the assertion declares. It writes the call's
// record when its seat records calls, and then sends the failure's record
// to a seat that satisfies [Reporter], and the writer's text of it to any
// other seat, through Fatalf under [Fatal] and Errorf under [Soft].
//
// The record's location is the innermost frame of the caller's code, which
// [CallerWhere] states, so an assertion reports the caller's line however
// many frames of this module are between them. Under [Fatal] it may not
// return.
//
// # Allocation contract
//
// A call without detail allocates 4 times on a seat that keeps the record
// and its text, as a seat of internal/matchertest does.
func Fail(seat Seat, mode Mode, assertion, contract string, detail map[string]any) {
	seat.Helper()
	Running{seat: seat}.Fail(mode, assertion, contract, detail)
}

// Fault reports that a call of assertion ended without a verdict, because
// err refused an argument or the environment. It writes the call's record,
// with the writer's text of err, when its seat records calls. It then
// passes err to a seat that satisfies [FaultReporter], and sends the text
// to any other seat through Fatalf. It does so under both modes: the call
// checked nothing, and a test that ran on would meet err again. It may not
// return.
//
// # Allocation contract
//
// A call of a fault without a path allocates 4 times on a seat that keeps
// the fault and its text, as a seat of internal/matchertest does.
func Fault(seat Seat, mode Mode, assertion, contract string, err error) {
	seat.Helper()
	Running{seat: seat}.Fault(mode, assertion, contract, err)
}

// Running is a call of an assertion that runs a body, such as Eventually:
// its record took its number before the calls of its body, and it reports
// its verdict when it ends. A Running that Begin did not return is a call
// that runs no body, and its verdict writes the record of an ordinary call.
type Running struct {
	// seat is the seat of the call.
	seat Seat
	// slot is the slot of the call's record, and nil for a call that
	// either runs no body or is not recorded.
	slot *record.Slot
	// body reports whether the call runs a body.
	body bool
}

// Begin starts a call on seat that runs a body. The call's record takes its
// number now, before the calls of its body, and the seat of each run of
// the body hands its calls to [Running.Slot].
//
// # Allocation contract
//
// Begin allocates nothing for a seat whose calls are not recorded.
func Begin(seat Seat) Running {
	return Running{seat: seat, slot: record.Begin(seat), body: true}
}

// Slot returns the slot that the seats of the body's runs hand their calls
// to, and nil for a call that is not recorded.
//
// # Allocation contract
//
// Slot allocates nothing.
func (r Running) Slot() *record.Slot {
	return r.slot
}

// Pass reports that the call passed, and marks its frame, as [Pass] does.
//
// # Allocation contract
//
// Pass allocates nothing for a call that is not recorded.
func (r Running) Pass(mode Mode, assertion, contract string) {
	if _, err := record.On(); err != nil {
		r.seat.Helper()
	}
	if !r.switched() {
		return
	}
	c := record.Call{Assertion: assertion, Contract: contract, Verdict: record.Pass, Aborting: mode == Fatal}
	r.write(c, nil, nil)
}

// Fail reports that the call failed with detail, as [Fail] does.
//
// # Allocation contract
//
// Fail allocates what [Fail] allocates.
func (r Running) Fail(mode Mode, assertion, contract string, detail map[string]any) {
	r.seat.Helper()
	if !r.switched() {
		return
	}
	where := site()
	r.write(record.Call{
		Assertion: assertion, Contract: contract, Verdict: record.Fail, Aborting: mode == Fatal,
		Where: record.Where(where),
	}, detail, nil)
	r.report(mode, Failure{Assertion: assertion, Contract: contract, Detail: detail, Where: where})
}

// PassRun reports that the call passed, as [Running.Pass] does, at where.
// The call's record states the JSON object that run encodes, the detail of
// the run that the call made, which a property states on every verdict. A
// call that is not recorded does not encode it.
//
// # Allocation contract
//
// PassRun allocates nothing for a call that is not recorded.
func (r Running) PassRun(mode Mode, assertion, contract string, where Where, run json.Marshaler) {
	r.seat.Helper()
	if !r.switched() {
		return
	}
	r.write(record.Call{
		Assertion: assertion, Contract: contract, Verdict: record.Pass, Aborting: mode == Fatal,
		Where: record.Where(where),
	}, nil, run)
}

// FailRun reports that the call failed with the record f, as [Running.Fail]
// does with the record that it builds. The call's record states the JSON
// object that run encodes, the detail of the run that the call made, in
// place of the typed literals of f's detail.
//
// # Allocation contract
//
// A call of a record without detail allocates twice on a seat that keeps
// the record and its text, as a seat of internal/matchertest does.
func (r Running) FailRun(mode Mode, f Failure, run json.Marshaler) {
	r.seat.Helper()
	if !r.switched() {
		return
	}
	r.write(record.Call{
		Assertion: f.Assertion, Contract: f.Contract, Verdict: record.Fail, Aborting: mode == Fatal,
		Where: record.Where(f.Where),
	}, nil, run)
	r.report(mode, f)
}

// report sends f to a seat that satisfies [Reporter], and the writer's text
// of it to any other seat, through Fatalf under [Fatal] and Errorf under
// [Soft].
func (r Running) report(mode Mode, f Failure) {
	r.seat.Helper()
	if reporter, ok := r.seat.(Reporter); ok {
		reporter.Report(f, mode == Fatal)
		return
	}
	text := writer.Failure(f)
	if mode == Soft {
		r.seat.Errorf("%s", text)
		return
	}
	r.seat.Fatalf("%s", text)
}

// Fault reports that the call ended without a verdict because of err, as
// [Fault] does.
//
// # Allocation contract
//
// Fault allocates what [Fault] allocates.
func (r Running) Fault(mode Mode, assertion, contract string, err error) {
	r.seat.Helper()
	if !r.switched() {
		return
	}
	r.write(record.Call{
		Assertion: assertion, Contract: contract, Verdict: record.Error, Aborting: mode == Fatal,
		Error: writer.Fault(err),
	}, nil, nil)
	r.end(err)
}

// end ends the call with err, a fault, as [End] ends a call.
func (r Running) end(err error) {
	r.seat.Helper()
	End(r.seat, err)
}

// switched reports whether the process's switch of recording states a
// value. When it does not, switched marks its frame and ends the call with
// the switch's fault.
func (r Running) switched() bool {
	if _, err := record.On(); err != nil {
		r.seat.Helper()
		r.end(err)
		return false
	}
	return true
}

// write writes c, the record of the call, into the slot of a call that runs
// a body, and into the Calls of the seat for any other call. The record
// states the JSON that run encodes for a call that made a run, and for any
// other failure the typed literals of detail, the failure's detail. A call
// that is not recorded builds nothing.
func (r Running) write(c record.Call, detail map[string]any, run json.Marshaler) {
	var calls *record.Calls
	if r.body {
		if r.slot == nil {
			return
		}
	} else if calls = record.Of(r.seat); calls == nil {
		return
	}
	if c.Where == (record.Where{}) {
		c.Where = record.Where(site())
	}
	if run != nil {
		c.Detail = runDetail(run)
	} else if c.Verdict == record.Fail {
		c.Detail = detailOf(detail)
	}
	if r.body {
		r.slot.Write(c)
		return
	}
	record.Add(calls, c)
}

// runDetail returns the JSON that run encodes, and the opaque literal of
// the error's text for a run whose detail cannot be encoded, so that the
// record is valid JSON.
func runDetail(run json.Marshaler) json.RawMessage {
	raw, err := run.MarshalJSON()
	if err != nil {
		return literal.Opaque(err.Error())
	}
	return raw
}

// detailOf returns the JSON object of a failure's detail: each field's
// typed literal, which [literal.Detail] states.
func detailOf(detail map[string]any) json.RawMessage {
	fields := make(map[string]json.RawMessage, len(detail))
	for name, value := range detail {
		fields[name] = literal.Detail(value)
	}
	// Marshal returns no error for a map of JSON values that Detail wrote.
	raw, _ := json.Marshal(fields)
	return raw
}
