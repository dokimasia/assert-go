// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
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
// A case that calls [Case.Skipf] before the run finds a failing case ends
// the run as [Skipped]. The run does not report a record, and skips the
// first seat of the seat's chain that has Skipf, such as a *testing.T. A
// chain without Skipf receives nothing.
//
// Under the campaign profile, which DOKIMI_ASSERT_PROP_PROFILE=campaign
// names, ForAll runs a campaign instead of a run, for as long as
// DOKIMI_ASSERT_PROP_BUDGET states in whole seconds of real time. A
// testing/synctest bubble does not fake that time. It tries the case of Draws and the stored cases, and then explores: a
// valid case that counts a new label, records a new fingerprint or records
// a better score with [Case.Target] joins the campaign's pool, and three
// cases in four mutate a member of the pool. Each failure of an identity of
// its own is shrunk, explained and stored as the campaign finds it. The
// record states the first failure, with every later one among its others,
// or the outcome that a run's last check decides over every valid case. A
// [Hermetic] run and a run in a test binary that a mutation run
// instrumented run no campaign.
//
// The record's detail states the eleven fields of the definition:
//
//   - outcome, an [Outcome].
//   - cases and rejected, the counts as int.
//   - seed, the run's seed as a decimal string.
//   - counterexample, a []Entry of the failing case's draws and steps, and
//     failure, its assert.Failure, for a counterexample and a flaky replay.
//   - choices, the replay token, and others, a []Other, for a
//     counterexample.
//   - divergence, a *Divergence, and coverage, a *Shortfall.
//   - reason, the reason of a skipped run as a string.
//
// A field that the outcome does not use is nil. The first [assert.Reporter]
// of the seat's chain, which [assert.TB] states, receives the record. A
// chain without one receives the failing case's notes in its log and then
// the record's sentence through Fatalf.
//
// The call's record states the detail of the run on a pass and on a skip as
// well, and the calls of each case that a run on one worker runs are
// recorded under it, with the phase of the case.
//
// [OnResult] hands the [Result] of the run to a function before the call
// reports its verdict. The result has the values of the eleven fields.
//
// ForAll ends the call with a fault, without a run, for a profile other than
// default, ci and campaign, a seed variable that is no decimal number below
// 2^64, a campaign's budget that is no whole number of seconds above 0, a
// token to replay that no encoder writes, entries of Draws that are no array
// of draw entries and step entries, a seat under [Around] without a test, a
// damaged file in the store, and a second property of the test with the
// same contract and store. It ends the call with a fault before any other
// case for a draw or a machine of the case of Draws that refuses its entry,
// and the fault is at the entry's label, value or step. It ends the call
// with a fault at Around, in place of the run's record, for a wrapper that
// returned without calling run or that called run a second time.
//
// A fault that ends a call in the body, such as the fault of [assert.Fault]
// on the case, ends the run in every phase: before the run finds a failing
// case, and in the replay, the shrink and the explanation of one. ForAll
// then ends its call with that fault, as the error it is, and stores no
// entry. A fault that the body notes passes to the seat as a noted fault.
//
// Under Around, the wrapper receives the first seat of the seat's chain that
// is a *testing.T, the test of an [assert.Recorder] of
// [assert.Recorder.WithTest], or the test of the assertion whose check runs
// on the seat, such as the check of [assert.Rejects]. A seat without one
// ends the call with a fault.
//
// It writes an entry for each failure of a
// counterexample to the store, unless a file of the entry's name exists or
// a mutation run instrumented the test binary, and logs the fault of a
// store that cannot keep it.
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
// of p's operation for a seat under Around without a test, a second property
// of the test with p's contract and store, a damaged file in the store, a
// wrapper of Around that misused its run, and a case of Draws that refuses an
// entry. It ends the call with the fault of a case that a fault ended.
func (p property) run(tb assert.TB, run matcher.Running, body func(*Case)) {
	tb.Helper()
	setup, ok := p.runsOn(tb, body)
	if !ok {
		p.fault(tb, run, fault.In(p.op, fault.At(fault.New("the seat has no *testing.T"),
			fault.Field(aroundOption))))
		return
	}
	if !claim(tb, p.dir, p.contract) {
		p.fault(tb, run, duplicate(p.op, p.dir, p.contract))
		return
	}
	s := setup.settings
	s.Slot, s.Seat = run.Slot(), tb
	if p.replaying {
		r := engine.RunReplay(setup.body, s, p.replay, record.Token)
		if err := setup.wrapper.fault(p.op); err != nil {
			p.fault(tb, run, err)
			return
		}
		p.report(tb, run, r)
		return
	}
	stored, err := p.load()
	if err != nil {
		p.fault(tb, run, err)
		return
	}
	s.Stored = storedChoices(stored)
	var r engine.Result
	var saved []error
	if s.Budget > 0 {
		r, saved = p.campaign(setup.body, s)
	} else {
		r = engine.Run(setup.body, s)
	}
	if err := setup.wrapper.fault(p.op); err != nil {
		p.fault(tb, run, err)
		return
	}
	if r.Refused != nil {
		p.fault(tb, run, fault.In(p.op, fault.At(r.Refused, fault.Field(drawsOption))))
		return
	}
	if r.Outcome == engine.Counterexample && s.Budget == 0 {
		saved = p.save(r)
	}
	faults := append(p.storeFaults(stored, r), saved...)
	for _, err := range faults {
		matcher.NoteFault(tb, err)
	}
	p.report(tb, run, r)
}

// campaign runs a campaign of cases under s, and returns its result and the
// faults of the store of each failure, which it stores as the campaign
// concludes it.
func (p property) campaign(cases engine.Body, s engine.Settings) (engine.Result, []error) {
	var saved []error
	s.Concluded = func(found engine.Result) { saved = append(saved, p.save(found)...) }
	return engine.Campaign(cases, s), saved
}
