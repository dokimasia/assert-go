// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"context"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/record"
)

// forAllOp is the operation of ForAll, which names its faults.
const forAllOp = "prop.ForAll"

// duplicate returns the fault of a property whose test has a second
// property of its contract in the store dir, which would share its stored
// cases.
func duplicate(op, dir, contract string) error {
	return fault.In(op, fault.At(fault.New(
		"two properties of the test have the contract %q, and would share their stored cases", contract),
		fault.Field(dir)))
}

// ForAll runs body against generated cases, and fails tb with one record of
// the assertion prop-for-all when the run does not pass. The record's
// contract is contract, and its location is the call of ForAll.
//
// A run tries the case of [Draws], then replays the property's stored cases
// oldest first, then the case whose every choice is its target, then random
// cases of the seed with a prefix case and an edge case after each, until
// [Cases] valid cases ran, the run tested every input of the domain, or ten
// times as many cases were generated. A failing case is replayed, shrunk to
// the smallest case that fails the same way, and explained. The options
// state the run's settings.
//
// A run that found no failing case fails when it rejected more than ten
// cases for every valid one, when no case requested an input, or when it
// refuted or left unmet a coverage requirement, checked in that order.
//
// The record's detail states the ten fields of the definition:
//
//   - outcome, an [Outcome].
//   - cases and rejected, the counts as int.
//   - seed, the run's seed as a decimal string.
//   - counterexample, a []Drawn, and failure, the failing case's
//     assert.Failure, for a counterexample and a flaky replay.
//   - choices, the replay token, and others, a []Other, for a
//     counterexample.
//   - divergence, a *Divergence, and coverage, a *Shortfall.
//
// A field that the outcome does not use is nil. An [assert.Reporter] seat
// receives the record. Any other seat receives the failing case's notes in
// its log and then the record's sentence through Fatalf.
//
// The call's record states the detail of the run on a pass as well, and the
// calls of each case that a run on one worker runs are recorded under it,
// with the phase of the case.
//
// ForAll ends the call with a fault, without a run, for a profile other than
// default and ci, a seed variable that is no decimal number below 2^64, a
// token to replay that no encoder writes, entries of Draws that are no array
// of labels and typed literals, a damaged file in the store, and a second
// property of the test with the same contract and store. It ends the call
// with a fault before any other case for a draw of the case of Draws that
// refuses its entry, and the fault names the draw's label. It writes an
// entry for each failure of a counterexample to the store, unless a file of
// the entry's name exists, and logs the fault of a store that cannot keep
// it.
//
// # Allocation contract
//
// A run allocates for each case the goroutine of its body, its recorder and
// the values its draws decode. The cases of a run on one worker reuse the
// storage of one record of choices, spans and draws. A passing run of 100
// cases that draw one integer each allocates 487 times, two of them for the
// closures that adapt the body to the engine's case and give each case a
// context. A failing run allocates its record and the runs of its shrink as
// well.
func ForAll(tb assert.TB, contract string, body func(*Case), opts ...Option) {
	tb.Helper()
	run := matcher.Begin(tb)
	p, err := newProperty(tb, forAllOp, contract, caller(), configure(opts))
	if err != nil {
		run.Fault(matcher.Fatal, forAllID, contract, err)
		return
	}
	p.run(tb, run, body)
}

// run runs body as the property p on tb, as the call run, and reports a
// run that does not pass, as [ForAll] states. It ends the call with a fault
// of p's operation for a second property of the test with p's contract and
// store, a damaged file in the store, and a draw of the case of Draws that
// refuses its entry.
func (p property) run(tb assert.TB, run matcher.Running, body func(*Case)) {
	tb.Helper()
	if !claim(tb, p.dir, p.contract) {
		p.fault(tb, run, duplicate(p.op, p.dir, p.contract))
		return
	}
	cases := bodyOf(contextOf(tb), body)
	s := p.settings
	s.Slot = run.Slot()
	if p.replaying {
		p.report(tb, run, engine.RunReplay(cases, s, p.replay, record.Token))
		return
	}
	stored, err := p.load()
	if err != nil {
		p.fault(tb, run, err)
		return
	}
	s.Stored = storedChoices(stored)
	r := engine.Run(cases, s)
	if r.Refused != nil {
		p.fault(tb, run, fault.In(p.op, fault.At(r.Refused, fault.Field(drawsOption))))
		return
	}
	faults := p.storeFaults(stored, r)
	if r.Outcome == engine.Counterexample {
		faults = append(faults, p.save(r)...)
	}
	for _, err := range faults {
		matcher.NoteFault(tb, err)
	}
	p.report(tb, run, r)
}

// contextOf returns the context of tb when tb states one, as a *testing.T
// does, and context.Background() for a seat without one, such as an
// [assert.Recorder].
func contextOf(tb assert.TB) context.Context {
	if seat, ok := tb.(interface{ Context() context.Context }); ok {
		return seat.Context()
	}
	return context.Background()
}
