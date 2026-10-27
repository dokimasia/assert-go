// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"context"
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/internal/wrap"
)

// callerSkip is the number of frames that [caller] skips: runtime.Callers,
// caller, and the exported function that calls it. The next frame is the
// call of that function.
const callerSkip = 3

// The names at the front of the path of a fault in what an option or an
// entry states.
const (
	// drawsOption is the option that states the entries of a case.
	drawsOption = "Draws"
	// aroundOption is the option that runs each run of the body inside a
	// wrapper.
	aroundOption = "Around"
	// valueMember is the member of an entry of Draws that states its value.
	valueMember = "value"
	// clientMember is the member of a step entry of Draws that states its
	// client.
	clientMember = "client"
)

// The reasons of the faults of a wrapper of Around that does not call its
// run once.
const (
	// skippedReason is the reason of a wrapper that returned without calling
	// run.
	skippedReason = "the wrapper returned without calling run"
	// repeatedReason is the reason of a wrapper that called run a second
	// time.
	repeatedReason = "the wrapper called run a second time"
)

// property is one call of [ForAll], [Fuzz] or a property form: the
// operation that names its faults, the assertion of its record, the
// contract it checks, the call's program counter, the settings of its runs,
// the generators of types that its options state, the token it replays,
// the directory of its store and the wrapper of its runs. A property is a
// value that its runs do not change.
type property struct {
	// op is the operation of the call: prop.ForAll, prop.Fuzz, or a form's.
	op string
	// assertion is the assertion of the call's record: prop-for-all, or a
	// form's id.
	assertion string
	// contract is the property's contract.
	contract string
	// pc is the program counter of the call, which a failing run resolves
	// to the call's location.
	pc uintptr
	// settings are the settings of the call's runs, without stored cases.
	settings engine.Settings
	// generators are the generators of types that Using states, which the
	// settings pass to every case, and nil for a call that states none.
	generators *generators
	// replay are the choices that the token to replay records, when
	// replaying is set.
	replay []choice.Choice
	// dir is the directory of the call's store, and empty for a call without
	// one.
	dir string
	// around is the wrapper of each run of the body, and nil for a call
	// without Around.
	around func(*testing.T, func(*testing.T))
	// replaying reports whether the call replays a token.
	replaying bool
}

// newProperty returns the property of the operation op of a call at pc on
// tb of contract under c, whose record states prop-for-all, and closes the
// registry. Its runs read the clock of tb and the generators of types that
// c states.
//
// It returns a fault of op for a profile other than default, ci and
// campaign, for a seed variable that is no decimal number below 2^64, for a
// campaign's budget that is no whole number of seconds above 0, for a token
// to replay that no encoder writes, and for entries of Draws that are no
// array of draw entries and step entries.
func newProperty(tb assert.TB, op, contract string, pc uintptr, c config) (property, error) {
	registrations.close()
	seed, err := seedOf(c, contract)
	if err != nil {
		return property{}, fault.In(op, err)
	}
	budget, err := budgetOf(c, op)
	if err != nil {
		return property{}, fault.In(op, err)
	}
	var entries []engine.Entry
	if c.drawn {
		if entries, err = drawEntries(c.draws); err != nil {
			return property{}, fault.In(op, err)
		}
	}
	gs := generatorsOf(c.using, nil)
	p := property{
		op:        op,
		assertion: forAllID,
		contract:  contract,
		pc:        pc,
		settings: engine.Settings{
			Seed:         seed,
			Cases:        c.cases,
			MaxChoices:   c.maxChoices,
			Requirements: c.requirements,
			Shrink:       c.shrink,
			ShrinkTime:   c.shrinkTime,
			Explain:      c.explain,
			Clock:        matcher.ClockOf(tb),
			Workers:      c.workers,
			Draws:        entries,
			Budget:       budget,
			Using:        gs.engineUsing(),
		},
		generators: gs,
		dir:        directoryOf(tb, c),
		around:     c.around,
	}
	if tok, source, ok := tokenOf(c); ok {
		if p.replay, err = token.Decode(tok); err != nil {
			return property{}, fault.In(op, fault.At(err, fault.Field(source)))
		}
		p.replaying = true
	}
	return p, nil
}

// runSetup is the settings and the body of the runs of a property on a
// seat, with the wrapper of each run.
type runSetup struct {
	// settings are the settings of the runs.
	settings engine.Settings
	// body is the engine's body of the runs.
	body engine.Body
	// wrapper runs each run of the body inside the wrapper of Around, and is
	// nil for a property without one.
	wrapper *wrapper
}

// runsOn returns the setup of the runs of the property p on tb. Without
// Around, each case derives its context from the context of tb. Under
// Around, each run of the body runs inside the wrapper, which receives the
// test that [testOf] finds for tb. It reports false under Around for a seat
// without one.
func (p property) runsOn(tb assert.TB, body func(*Case)) (runSetup, bool) {
	if p.around == nil {
		return runSetup{settings: p.settings, body: bodyOf(matcher.ContextOf(tb), body)}, true
	}
	t, ok := testOf(tb)
	if !ok {
		return runSetup{}, false
	}
	w := &wrapper{wrap: p.around, t: t}
	s := p.settings
	s.Around = w.run
	return runSetup{settings: s, body: caseBody(body), wrapper: w}, true
}

// tested is a seat that runs in a test without wrapping it, as a Recorder
// of [assert.Recorder.WithTest] does.
type tested interface {
	// Test returns the test of the seat, and nil for none.
	Test() assert.TB
}

// nested is the seat of the body of an assertion, such as the check of
// [assert.Rejects], which runs in the test of the assertion's seat.
type nested interface {
	// Outer returns the seat of the assertion.
	Outer() matcher.Seat
}

// testOf returns the test that the wrapper of Around receives on seat: the
// first seat of the chain of seat that is a *testing.T, or the test that
// testOf finds for the test of a tested seat or for the outer seat of a
// nested seat. It reports false for a seat without one. The search does not
// unwrap a tested or a nested seat, so a property on it keeps its store, its
// log and its skips to that seat.
func testOf(seat any) (*testing.T, bool) {
	for ; seat != nil; seat = wrap.Unwrap(seat) {
		switch s := seat.(type) {
		case *testing.T:
			return s, true
		case tested:
			if t := s.Test(); t != nil {
				return testOf(t)
			}
		case nested:
			return testOf(s.Outer())
		}
	}
	return nil, false
}

// wrapper runs each run of a property's body inside the wrapper of
// [Around], which receives the test t. It keeps the reason of the first
// call of the wrapper that returned without calling run or that called run a
// second time. From then on it calls the wrapper no more, and each later run
// runs no body.
//
// # Concurrency
//
// Its methods are safe for concurrent use.
type wrapper struct {
	// wrap is the wrapper of Around.
	wrap func(*testing.T, func(*testing.T))
	// t is the test that wrap receives.
	t *testing.T
	// misused is the reason of the first misuse of run, and nil before one.
	misused atomic.Pointer[string]
}

// run is the engine's Around. It calls wrap with the test and a function
// that calls run once with the context of the test that wrap hands it. A
// second call of that function, and a call after wrap has returned, return
// without a call of run. After the first misuse, run calls wrap no more.
func (w *wrapper) run(run func(context.Context)) {
	if w.misused.Load() != nil {
		return
	}
	var mu sync.Mutex
	calls, open := 0, true
	w.wrap(w.t, func(t *testing.T) {
		t.Helper()
		mu.Lock()
		calls++
		first := calls == 1 && open
		mu.Unlock()
		if first {
			run(t.Context())
		}
	})
	mu.Lock()
	n := calls
	open = false
	mu.Unlock()
	if n == 0 {
		w.keep(skippedReason)
	}
	if n > 1 {
		w.keep(repeatedReason)
	}
}

// keep keeps reason as the reason of the first misuse.
func (w *wrapper) keep(reason string) {
	w.misused.CompareAndSwap(nil, &reason)
}

// fault returns the fault of op at Around for the first misuse of run, and
// nil before one and for a nil wrapper.
func (w *wrapper) fault(op string) error {
	if w == nil {
		return nil
	}
	reason := w.misused.Load()
	if reason == nil {
		return nil
	}
	return fault.In(op, fault.At(fault.New("%s", *reason), fault.Field(aroundOption)))
}

// drawEntries returns the entries of Draws that text states: a JSON array
// of objects, each a step entry, which states the action of a step under
// "step", its client under "client" in a concurrent section and "drain":
// true in the drain, or a draw entry, which states a label and a typed
// literal. It returns a fault at Draws for other text, and at the entry for
// an entry that states no step and no label or no value, for a client below
// 0, and for a value that is no typed literal.
func drawEntries(text string) ([]engine.Entry, error) {
	var stated []struct {
		Label  *string         `json:"label"`
		Value  json.RawMessage `json:"value"`
		Step   *string         `json:"step"`
		Client *int            `json:"client"`
		Drain  bool            `json:"drain"`
	}
	if err := json.Unmarshal([]byte(text), &stated); err != nil {
		return nil, fault.At(fault.New("the entries are no JSON array of objects").Because(err),
			fault.Field(drawsOption))
	}
	entries := make([]engine.Entry, len(stated))
	for i, s := range stated {
		if s.Step != nil {
			step := engine.MachineStep{Action: *s.Step, Client: -1, Drain: s.Drain}
			if s.Client != nil && *s.Client < 0 {
				return nil, fault.At(fault.New("the client %d is below 0", *s.Client), fault.Field(drawsOption),
					fault.Index(i), fault.Field(clientMember))
			}
			if s.Client != nil {
				step.Client = *s.Client
			}
			entries[i] = engine.Entry{Step: &step}
			continue
		}
		if s.Label == nil || s.Value == nil {
			return nil, fault.At(fault.New("the entry states no step, and no label or no value"),
				fault.Field(drawsOption), fault.Index(i))
		}
		value, err := literal.Decode(s.Value)
		if err != nil {
			return nil, fault.At(err, fault.Field(drawsOption), fault.Index(i), fault.Field(valueMember))
		}
		entries[i] = engine.Entry{Label: *s.Label, Value: value}
	}
	return entries, nil
}

// load returns the entries of the property in its store, oldest first, with
// the fault of each file that the run skips. A property without a store has
// no entries.
//
// It returns a fault of the property's operation at the store's directory,
// whose cause states each damaged file of the store, or the error of the
// file system for a store that cannot be read.
func (p property) load() (store.Stored, error) {
	if p.dir == "" {
		return store.Stored{}, nil
	}
	stored, err := store.Load(p.dir, p.contract)
	if err != nil {
		return store.Stored{}, fault.In(p.op, fault.At(err, fault.Field(p.dir)))
	}
	return stored, nil
}

// save writes an entry for the failing case of r, a counterexample, and one
// for each of its other failures, found at the time of the property's
// clock. It returns a fault of the property's operation at the store's
// directory for each entry that the store cannot keep. A property without a
// store saves nothing, and so does a property in a test binary that a
// mutation run instrumented: a counterexample of one mutant would enter the
// module under test, and every later mutant's run would try it first. An
// example of values has no entry: an entry records choices, and the
// example's values are in the test's source.
func (p property) save(r engine.Result) []error {
	if p.dir == "" || matcher.Mutated() || r.Failing.Case.Valued() {
		return nil
	}
	found := p.settings.Clock.Now()
	var faults []error
	for _, e := range append([]engine.Execution{*r.Failing}, r.Others...) {
		if _, err := store.Save(p.dir, entryOf(p.contract, e, found)); err != nil {
			faults = append(faults, fault.In(p.op, fault.At(
				fault.New("the store keeps no case of %q", p.contract).Because(err), fault.Field(p.dir))))
		}
	}
	return faults
}

// report reports the verdict of the property's call run on tb, from r, the
// result of its run. A run that a fault of a case ended ends the call with
// that fault. A run that passed passes, and a run that skipped skips the
// first seat of the chain of tb that has Skipf with its reason. The record of
// either states the detail of the run. A run that did none of these writes
// the notes of its failing case into the log of tb, and then fails with its
// record. The first [assert.Reporter] of the chain of tb receives the record
// as an aborting failure, and a chain without one receives its sentence
// through Fatalf.
func (p property) report(tb assert.TB, run matcher.Running, r engine.Result) {
	tb.Helper()
	if r.Fault != nil {
		p.fault(tb, run, r.Fault)
		return
	}
	if r.Outcome == engine.Passed || r.Outcome == engine.Skipped {
		var where assert.Where
		var detail json.Marshaler
		if run.Slot() != nil {
			where, detail = whereOf(p.pc), detailOf(r)
		}
		if r.Outcome == engine.Skipped {
			run.SkipRun(matcher.Fatal, p.assertion, p.contract, where, detail, r.Reason)
			return
		}
		run.PassRun(matcher.Fatal, p.assertion, p.contract, where, detail)
		return
	}
	for _, n := range notesOf(r) {
		matcher.Note(tb, n)
	}
	d := detailOf(r)
	run.FailRun(matcher.Fatal, assert.Failure{
		Assertion: p.assertion, Contract: p.contract, Detail: d.fields(),
		Where: whereOf(p.pc),
	}, d)
}

// fault ends the property's call run on tb with err, a fault that refused
// an argument or the environment.
func (p property) fault(tb assert.TB, run matcher.Running, err error) {
	tb.Helper()
	run.Fault(matcher.Fatal, p.assertion, p.contract, err)
}

// notesOf returns the notes of the failing case of r, and none for a run
// without one.
func notesOf(r engine.Result) []string {
	if r.Failing == nil {
		return nil
	}
	return r.Failing.Case.Notes()
}

// caller returns the program counter of the call of the exported function
// that calls caller. It allocates nothing, and [whereOf] resolves the
// counter when a run reports a failure.
func caller() uintptr {
	var pcs [1]uintptr
	runtime.Callers(callerSkip, pcs[:])
	return pcs[0]
}

// whereOf returns the location of the call at pc, which [caller] returned.
func whereOf(pc uintptr) assert.Where {
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	return assert.Where{File: frame.File, Line: frame.Line}
}
