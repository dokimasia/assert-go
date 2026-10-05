// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"runtime"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/prop/token"
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
	// valueMember is the member of an entry of Draws that states its value.
	valueMember = "value"
	// clientMember is the member of a step entry of Draws that states its
	// client.
	clientMember = "client"
)

// property is one call of [ForAll], [Fuzz] or a property form: the
// operation that names its faults, the assertion of its record, the
// contract it checks, the call's program counter, the settings of its runs,
// the token it replays and the directory of its store. A property is a
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
	// replay are the choices that the token to replay records, when
	// replaying is set.
	replay []choice.Choice
	// replaying reports whether the call replays a token.
	replaying bool
	// dir is the directory of the call's store, and empty for a call without
	// one.
	dir string
}

// newProperty returns the property of the operation op of a call at pc on
// tb of contract under c, whose record states prop-for-all, and closes the
// registry. Its runs read the clock of tb.
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
	budget, err := budgetOf(op)
	if err != nil {
		return property{}, fault.In(op, err)
	}
	var entries []engine.Entry
	if c.drawn {
		if entries, err = drawEntries(c.draws); err != nil {
			return property{}, fault.In(op, err)
		}
	}
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
		},
		dir: directoryOf(tb, c),
	}
	if tok, source, ok := tokenOf(c); ok {
		if p.replay, err = token.Decode(tok); err != nil {
			return property{}, fault.In(op, fault.At(err, fault.Field(source)))
		}
		p.replaying = true
	}
	return p, nil
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
// store saves nothing.
func (p property) save(r engine.Result) []error {
	if p.dir == "" {
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
// result of its run. A run that passed passes, and its record states the
// detail of the run. A run that did not pass writes the notes of its
// failing case into the log of tb, and then fails with its record: an
// [assert.Reporter] seat receives the record as an aborting failure, and any
// other seat its sentence through Fatalf.
func (p property) report(tb assert.TB, run matcher.Running, r engine.Result) {
	tb.Helper()
	if r.Outcome == engine.Passed {
		var where assert.Where
		var detail json.Marshaler
		if run.Slot() != nil {
			where, detail = whereOf(p.pc), detailOf(r)
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
