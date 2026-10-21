// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
)

// fuzzOp is the operation of Fuzz, which names its faults.
const fuzzOp = "prop.Fuzz"

// Fuzz registers body as the fuzz target of f. go test -fuzz generates the
// inputs. go test without -fuzz runs the seed corpus alone: the entries that
// f.Add states and the files under testdata/fuzz/<FuzzName>. Fuzz generates
// no case of its own, so a property that runs under go test without -fuzz
// is checked by [ForAll] in a test, with the same body and contract.
//
// Before it registers the target, Fuzz replays the property's stored cases,
// oldest first, from the store of the fuzz test, as ForAll replays them,
// and fails f with the record of the first that fails, as found. A failing
// replay registers no target. The record of the call on f counts the stored
// cases that ran, the failing one's predecessors included, and states the
// calls of each under the phase stored. Fuzz logs the fault of each stored
// case that decodes to other values than its entry records, as ForAll does.
// Fuzz runs no campaign, also under the campaign profile. [Replay],
// DOKIMI_ASSERT_PROP_REPLAY, [Draws], [Cases] and [Require] apply to ForAll
// and the property forms alone.
//
// Each input's bytes decode into the choices of one case by the definition's
// bridge rules, so every input is a valid case. A body that draws one byte
// string without a maximum size reads its length from the first two bytes,
// little-endian, and the string from the bytes after them, up to the last.
// The bridge decodes each entry of the seed corpus as it decodes a fuzzer's
// input, so an entry states the bytes of a case's choices and not a value.
//
// A failing input's case is replayed, shrunk and explained as [ForAll]
// does with a failing case. Fuzz writes the counterexample to the store,
// unless a mutation run instrumented the test binary, and reports it
// through the input's *testing.T with its replay token.
//
// Each input is a call of its own, whose record states the calls of the
// input's case under the phase fuzz. The record of a passing input counts
// one valid or one rejected case, and the record of a failing input counts
// none. The report, the notes of the failing case and the faults of the
// store are written to the test's output without a source location. The
// fuzzing machinery calls the target through reflect, so no frame of the
// caller's code is on the stack.
//
// Fuzz ends the call on f with a fault, without registering the target, for
// each fault for which ForAll ends its call without a run. It ends the call
// of an input with a fault at Around, in place of its record, for a wrapper
// that returned without calling run or that called run a second time.
//
// Under [Around], the wrapper runs each run of the case of an input, its
// replay, its shrink and its explanation, with the test of the input. A
// *testing.F has no *testing.T for the wrapper, so the call on f replays no
// stored case, and notes the faults of the files of the store alone. The
// first input that the process runs replays the stored cases before its own
// case, as a call of its own on its test. A failing stored case fails that
// input with its record, as found, and the case of the input does not run.
func Fuzz(f *testing.F, contract string, body func(*Case), opts ...Option) {
	f.Helper()
	run := matcher.Begin(f)
	p, err := newProperty(f, fuzzOp, contract, caller(), configure(opts))
	if err != nil {
		run.Fault(matcher.Fatal, forAllID, contract, err)
		return
	}
	if !claim(f, p.dir, p.contract) {
		p.fault(f, run, duplicate(p.op, p.dir, p.contract))
		return
	}
	stored, err := p.load()
	if err != nil {
		p.fault(f, run, err)
		return
	}
	var first store.Stored
	if p.around != nil {
		first, stored = store.Stored{Entries: stored.Entries}, store.Stored{Skipped: stored.Skipped}
	}
	p.replayStored(f, run, runSetup{settings: p.settings, body: bodyOf(f.Context(), body)}, stored)
	var once sync.Once
	f.Fuzz(func(t *testing.T, data []byte) {
		seat := inputSeat{T: t}
		setup, _ := p.runsOn(t, body)
		if len(first.Entries) > 0 {
			once.Do(func() { p.replayStored(seat, matcher.Begin(seat), setup, first) })
		}
		p.input(seat, setup, data)
	})
}

// replayStored replays the entries of stored on seat, as the call run, with
// the runs of setup, and reports the run, as [Fuzz] states. It notes the
// fault of each file of stored that the run skips, and of each entry whose
// case decodes to other values than the entry records. A report that does
// not pass ends the call, and so does a wrapper of Around that misused its
// run.
func (p property) replayStored(seat assert.TB, run matcher.Running, setup runSetup, stored store.Stored) {
	seat.Helper()
	s := setup.settings
	s.Slot = run.Slot()
	s.Stored = storedChoices(stored)
	r := engine.RunStored(setup.body, s)
	if err := setup.wrapper.fault(p.op); err != nil {
		p.fault(seat, run, err)
		return
	}
	for _, err := range p.storeFaults(stored, r) {
		matcher.NoteFault(seat, err)
	}
	p.report(seat, run, r)
}

// input runs the case that data decodes to as a call of the property on
// seat, the seat of the input's test, with the runs of setup, and reports
// the call's verdict. A failing case is concluded as ForAll concludes one,
// and the failures of a counterexample are written to the store. A wrapper
// of Around that misused its run ends the call with a fault.
func (p property) input(seat inputSeat, setup runSetup, data []byte) {
	seat.Helper()
	run := matcher.Begin(seat)
	s := setup.settings
	s.Slot = run.Slot()
	e := engine.Bridge(setup.body, data, s)
	r := engine.Result{Outcome: engine.Passed, Cases: 1, Seed: s.Seed}
	if e.Status == engine.CaseRejected {
		r.Cases, r.Rejected = 0, 1
	}
	if e.Status == engine.CaseFailed {
		r = engine.Conclude(setup.body, s, e)
	}
	if err := setup.wrapper.fault(p.op); err != nil {
		p.fault(seat, run, err)
		return
	}
	if r.Outcome == engine.Counterexample {
		for _, err := range p.save(r) {
			matcher.NoteFault(seat, err)
		}
	}
	p.report(seat, run, r)
}

// inputSeat is the seat of the test of one fuzz input. It writes the
// failure and each note and fault of the input's call to the test's
// output, which states no source location, and writes call records through
// the test's Attr.
type inputSeat struct {
	*testing.T
}

// Fatalf writes the formatted text to the test's output and stops the
// test.
func (s inputSeat) Fatalf(format string, args ...any) {
	s.Logf(format, args...)
	s.FailNow()
}

// Logf writes the formatted text to the test's output.
func (s inputSeat) Logf(format string, args ...any) {
	fmt.Fprintf(s.Output(), format+"\n", args...)
}
