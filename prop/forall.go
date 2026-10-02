// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"context"
	"slices"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// duplicate is the problem of a test whose two properties share a contract,
// and with it their stored cases.
const duplicate = "prop: two properties of the test have the contract %q, and would share their stored cases"

// ForAll runs body against generated cases, and fails tb with one record of
// the assertion prop-for-all when the run does not pass. The record's
// contract is contract, and its location is the call of ForAll.
//
// A run replays the property's stored cases oldest first, then the case
// whose every choice is its target, then random cases of the seed with a
// prefix case and an edge case after each, until [Cases] valid cases ran,
// the run tested every input of the domain, or ten times as many cases were
// generated. A failing case is replayed, shrunk to the smallest case that
// fails the same way, and explained. The options state the run's settings.
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
// receives the record. Any other seat receives its sentence through Fatalf,
// with the failing case's notes.
//
// ForAll fails tb at once, without a run, for a profile other than default
// and ci, a seed variable that is no decimal number below 2^64, a token to
// replay that no encoder writes, a damaged file in the store, and a second
// property of the test with the same contract and store. It writes an entry
// for each failure of a counterexample to the store, unless a file of the
// entry's name exists, and logs a note on a store that cannot keep it.
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
	p, err := newProperty(tb, contract, caller(), configure(opts))
	if err != nil {
		tb.Fatalf("%v", err)
		return
	}
	if !claim(tb, p.dir, contract) {
		tb.Fatalf(duplicate, contract)
		return
	}
	run := bodyOf(contextOf(tb), body)
	if p.replaying {
		p.report(tb, engine.RunReplay(run, p.settings, p.replay))
		return
	}
	stored, err := p.load()
	if err != nil {
		tb.Fatalf("%v", err)
		return
	}
	s := p.settings
	for _, e := range stored.Entries {
		s.Stored = append(s.Stored, e.Choices)
	}
	r := engine.Run(run, s)
	notes := slices.Concat(stored.Skipped, differences(stored.Entries, r.Stored))
	if r.Outcome == engine.Counterexample {
		notes = append(notes, p.save(r)...)
	}
	note(tb, notes)
	p.report(tb, r)
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
