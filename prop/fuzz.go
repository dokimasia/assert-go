// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"flag"
	"fmt"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// fuzzOp is the operation of Fuzz, which names its faults.
const fuzzOp = "prop.Fuzz"

// Fuzz checks the property that body states on f, and registers body as the
// fuzz target of f, so one declaration checks the property under go test and
// serves the fuzzer under go test -fuzz.
//
// In a test binary that does not fuzz, which the flag test.fuzz states, Fuzz
// first runs the property on f as [ForAll] runs it, with the same record,
// store and options, and fails f as ForAll fails its seat. A failing run
// registers no target. Fuzz runs no campaign, also under the campaign
// profile. go test then runs the seed corpus: the entries that f.Add states
// and the files under testdata/fuzz/<FuzzName>.
//
// In a test binary that fuzzes, Fuzz replays only the property's stored
// cases, oldest first, from the store of the fuzz test, and fails f with the
// record of the first that fails, as found. The record of the call on f
// counts the stored cases that ran, the failing one's predecessors included,
// and states the calls of each under the phase stored. Fuzz logs the fault
// of each stored case that decodes to other values than its entry records,
// as ForAll does. [Replay], DOKIMI_ASSERT_PROP_REPLAY, [Draws], [Cases] and
// [Require] apply to the run without fuzzing alone.
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
// each fault for which ForAll ends its call without a run.
func Fuzz(f *testing.F, contract string, body func(*Case), opts ...Option) {
	f.Helper()
	run := matcher.Begin(f)
	p, err := newProperty(f, fuzzOp, contract, caller(), configure(opts))
	if err != nil {
		run.Fault(matcher.Fatal, forAllID, contract, err)
		return
	}
	if fuzzing() {
		p.replayStored(f, run, body)
	} else {
		p.run(f, run, body)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		p.input(inputSeat{T: t}, bodyOf(t.Context(), body), data)
	})
}

// fuzzing reports whether the test binary fuzzes. The testing package
// registers the flag test.fuzz in every test binary, and -fuzz sets it.
func fuzzing() bool {
	return flag.Lookup("test.fuzz").Value.String() != ""
}

// replayStored replays the stored cases of the property p on f, as the call
// run, and reports the run, as [Fuzz] states for a test binary that fuzzes.
// A report that does not pass ends the call on f.
func (p property) replayStored(f *testing.F, run matcher.Running, body func(*Case)) {
	f.Helper()
	if !claim(f, p.dir, p.contract) {
		p.fault(f, run, duplicate(p.op, p.dir, p.contract))
		return
	}
	stored, err := p.load()
	if err != nil {
		p.fault(f, run, err)
		return
	}
	s := p.settings
	s.Slot = run.Slot()
	s.Stored = storedChoices(stored)
	r := engine.RunStored(bodyOf(f.Context(), body), s)
	for _, err := range p.storeFaults(stored, r) {
		matcher.NoteFault(f, err)
	}
	p.report(f, run, r)
}

// input runs the case that data decodes to as a call of the property on
// seat, the seat of the input's test, and reports the call's verdict. A
// failing case is concluded as ForAll concludes one, and the failures of a
// counterexample are written to the store.
func (p property) input(seat inputSeat, cases engine.Body, data []byte) {
	seat.Helper()
	run := matcher.Begin(seat)
	s := p.settings
	s.Slot = run.Slot()
	e := engine.Bridge(cases, data, s)
	if e.Status != engine.CaseFailed {
		r := engine.Result{Outcome: engine.Passed, Seed: s.Seed}
		if e.Status == engine.CaseRejected {
			r.Rejected = 1
		} else {
			r.Cases = 1
		}
		p.report(seat, run, r)
		return
	}
	r := engine.Conclude(cases, s, e)
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
