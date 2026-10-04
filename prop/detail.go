// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"strconv"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/text"
)

// The assertion of a run's record, and the names of its detail fields in
// the order the definition lists them.
const (
	// forAllID is the assertion of the record.
	forAllID = "prop-for-all"
	// outcomeField is how the run ended, an [Outcome].
	outcomeField = "outcome"
	// casesField is the number of valid cases, an int.
	casesField = "cases"
	// rejectedField is the number of rejected cases, an int.
	rejectedField = "rejected"
	// seedField is the run's seed as a decimal string.
	seedField = "seed"
	// counterexampleField is the failing case's draws, a []Drawn.
	counterexampleField = "counterexample"
	// failureField is the failing case's record, an assert.Failure.
	failureField = "failure"
	// choicesField is the failing case's replay token, a string.
	choicesField = "choices"
	// othersField is the run's other failures, a []Other.
	othersField = "others"
	// divergenceField is what differed in a flaky run, a *Divergence.
	divergenceField = "divergence"
	// coverageField is the requirement a run missed, a *Shortfall.
	coverageField = "coverage"
)

// The detail fields of the record of a case whose body panicked.
const (
	// panicField is the type of the panic's value.
	panicField = "panic"
	// stackField is the stack of the body's goroutine at the panic, as
	// runtime/debug.Stack formats it.
	stackField = "stack"
)

// runDetail is the detail of the record of a property's run: the ten fields
// of the definition. A field that the outcome does not use is nil:
//
//   - counterexample and failure are set for a run with a failing case: a
//     counterexample, and a flaky run whose replay differed.
//   - choices and others are set for a counterexample.
//   - divergence is set for a flaky run, and coverage for a run that missed
//     a coverage requirement.
type runDetail struct {
	// outcome is how the run ended.
	outcome Outcome
	// cases and rejected are the counts of the valid and the rejected cases.
	cases, rejected int
	// seed is the run's seed as a decimal string.
	seed string
	// counterexample are the failing case's draws.
	counterexample []Drawn
	// failure is the failing case's record.
	failure *assert.Failure
	// choices is the failing case's replay token.
	choices *string
	// others are the run's other failures.
	others []Other
	// divergence is the engine's divergence, whose sides the JSON states as
	// the corpus does.
	divergence *engine.Divergence
	// coverage is the requirement that the run missed.
	coverage *Shortfall
}

// detailOf returns the detail of the record of r.
func detailOf(r engine.Result) runDetail {
	d := runDetail{
		outcome:  Outcome(r.Outcome),
		cases:    r.Cases,
		rejected: r.Rejected,
		seed:     strconv.FormatUint(r.Seed, 10),
	}
	if r.Failing != nil {
		failure := failureOf(*r.Failing)
		d.counterexample, d.failure = counterexampleOf(*r.Failing, r.Explanation), &failure
	}
	if r.Outcome == engine.Counterexample {
		d.choices, d.others = &r.Token, othersOf(r.Others)
	}
	d.divergence = r.Divergence
	if r.Shortfall != nil {
		d.coverage = shortfallOf(*r.Shortfall)
	}
	return d
}

// fields returns the detail as a failure record states it: each field of
// the definition by its name, with its Go value, and nil for a field that
// the outcome does not use.
func (d runDetail) fields() map[string]any {
	detail := map[string]any{
		outcomeField:        d.outcome,
		casesField:          d.cases,
		rejectedField:       d.rejected,
		seedField:           d.seed,
		counterexampleField: nil,
		failureField:        nil,
		choicesField:        nil,
		othersField:         nil,
		divergenceField:     nil,
		coverageField:       nil,
	}
	if d.failure != nil {
		detail[counterexampleField], detail[failureField] = d.counterexample, *d.failure
	}
	if d.choices != nil {
		detail[choicesField], detail[othersField] = *d.choices, d.others
	}
	if d.divergence != nil {
		detail[divergenceField] = divergenceOf(*d.divergence)
	}
	if d.coverage != nil {
		detail[coverageField] = d.coverage
	}
	return detail
}

// detailJSON is the detail of a run as the call record of a property states
// it, in the order that the definition lists its fields.
type detailJSON struct {
	Outcome        Outcome         `json:"outcome"`
	Cases          int             `json:"cases"`
	Rejected       int             `json:"rejected"`
	Seed           string          `json:"seed"`
	Counterexample []Drawn         `json:"counterexample"`
	Failure        *assert.Failure `json:"failure"`
	Choices        *string         `json:"choices"`
	Others         []Other         `json:"others"`
	Divergence     *divergenceJSON `json:"divergence"`
	Coverage       *Shortfall      `json:"coverage"`
}

// divergenceJSON is a divergence as the call record of a property states
// it: a side is the bounds of a request, the text of a failure's identity,
// a fingerprint, or null.
type divergenceJSON struct {
	What     Difference `json:"what"`
	Index    int        `json:"index"`
	Recorded any        `json:"recorded"`
	Replayed any        `json:"replayed"`
}

// MarshalJSON returns the detail as the call record of a property states
// it, in the form of the definition's vectors: the counts as numbers, the
// seed and the token as strings, each drawn value as a typed literal, a
// failure as its failure record, the bounds of a request as an object, and
// null for a field that the outcome does not use.
func (d runDetail) MarshalJSON() ([]byte, error) {
	out := detailJSON{
		Outcome: d.outcome, Cases: d.cases, Rejected: d.rejected, Seed: d.seed,
		Counterexample: d.counterexample, Failure: d.failure, Choices: d.choices, Others: d.others,
		Coverage: d.coverage,
	}
	if v := d.divergence; v != nil {
		out.Divergence = &divergenceJSON{
			What: Difference(v.What), Index: v.Index, Recorded: sideOf(v.Recorded),
			Replayed: sideOf(v.Replayed),
		}
	}
	return json.Marshal(out)
}

// sideOf returns one side of the engine's divergence as the call record of
// a property states it: bounds as themselves, whose JSON is their corpus
// form, an identity as its text, and a fingerprint or nil as it is.
func sideOf(side any) any {
	switch side := side.(type) {
	case choice.Bounds:
		return side
	case engine.Identity:
		return identityText(side)
	}
	return side
}

// failureOf returns the record of a failed case: the first record that its
// body kept, or for a panic a record without an assertion whose contract is
// the panic's value, with the value's type and the stack in its detail.
func failureOf(e engine.Execution) assert.Failure {
	if e.Panic == nil {
		return e.Case.Failures()[0]
	}
	return assert.Failure{
		Contract: text.Sprintf("%v", e.Panic),
		Detail:   map[string]any{panicField: e.Identity.Panic, stackField: string(e.Stack)},
		Where:    e.Identity.Where,
	}
}
