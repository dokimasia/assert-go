// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"runtime"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/prop/token"
)

// callerSkip is the number of frames that [caller] skips to reach the call
// of the exported function that calls it: runtime.Callers, caller, and that
// function.
const callerSkip = 3

// logger is a seat with a log, as testing.TB has one.
type logger interface {
	// Logf formats its arguments as fmt.Sprintf does and adds the text to
	// the test's log.
	Logf(format string, args ...any)
}

// property is one call of [ForAll] or [Fuzz]: the contract it checks, the
// call's program counter, the settings of its runs, the token it replays
// and the directory of its store. A property is a value that its runs do
// not change.
type property struct {
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

// newProperty returns the property of a call at pc on tb of contract under
// c. Its runs read the clock of tb.
//
// It returns an error for a profile other than default and ci, for a seed
// variable that is no decimal number below 2^64, and for a token to replay
// that no encoder writes.
func newProperty(tb assert.TB, contract string, pc uintptr, c config) (property, error) {
	seed, err := seedOf(c, contract)
	if err != nil {
		return property{}, err
	}
	p := property{
		contract: contract,
		pc:       pc,
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
		},
		dir: directoryOf(tb, c),
	}
	if tok, ok := tokenOf(c); ok {
		if p.replay, err = token.Decode(tok); err != nil {
			return property{}, fmt.Errorf("prop: replay %q: %w", tok, err)
		}
		p.replaying = true
	}
	return p, nil
}

// load returns the entries of the property in its store, oldest first, with
// a note on each file that the run skips. A property without a store has no
// entries.
//
// It returns an error that states the name of each damaged file of the
// store, and the error of the file system for a store that cannot be read.
func (p property) load() (store.Stored, error) {
	if p.dir == "" {
		return store.Stored{}, nil
	}
	stored, err := store.Load(p.dir, p.contract)
	if err != nil {
		return store.Stored{}, fmt.Errorf("prop: the store %s of %q: %w", p.dir, p.contract, err)
	}
	return stored, nil
}

// save writes an entry for the failing case of r, a counterexample, and one
// for each of its other failures, found at the time of the property's
// clock. It returns a note on each entry that the store cannot keep. A
// property without a store saves nothing.
func (p property) save(r engine.Result) []string {
	if p.dir == "" {
		return nil
	}
	found := p.settings.Clock.Now()
	var notes []string
	for _, e := range append([]engine.Execution{*r.Failing}, r.Others...) {
		if _, err := store.Save(p.dir, entryOf(p.contract, e, found)); err != nil {
			notes = append(notes, fmt.Sprintf("prop: the store %s keeps no case of %q: %v", p.dir, p.contract, err))
		}
	}
	return notes
}

// report fails tb with the record of r when r did not pass. An
// [assert.Reporter] seat receives the record as an aborting failure. Any
// other seat receives its sentence through Fatalf, with the notes of the
// failing case.
func (p property) report(tb assert.TB, r engine.Result) {
	tb.Helper()
	if r.Outcome == engine.Passed {
		return
	}
	f := assert.Failure{Assertion: forAllID, Contract: p.contract, Detail: detailOf(r), Where: whereOf(p.pc)}
	if reporter, ok := tb.(assert.Reporter); ok {
		reporter.Report(f, true)
		return
	}
	var notes []string
	if r.Failing != nil {
		notes = r.Failing.Case.Notes()
	}
	tb.Fatalf("%s", render(f, notes))
}

// note adds each note to the log of tb, when tb has a log.
func note(tb assert.TB, notes []string) {
	tb.Helper()
	l, ok := tb.(logger)
	if !ok {
		return
	}
	for _, n := range notes {
		l.Logf("%s", n)
	}
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
