// Copyright Dokimasia B.V. 2026
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
	// counterexampleField is the failing case's entries, a []Entry.
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
	// reasonField is the reason of a skipped run, a string.
	reasonField = "reason"
)

// The detail fields of the record of a case whose body panicked.
const (
	// panicField is the type of the panic's value.
	panicField = "panic"
	// stackField is the stack of the body's goroutine at the panic, as
	// runtime/debug.Stack formats it.
	stackField = "stack"
)

// runDetail is the detail of the record of a property's run: the eleven
// fields of the definition, which the run's [Result] contains, and the
// engine's divergence of a flaky run.
type runDetail struct {
	// result is the result of the run.
	result Result
	// divergence is the engine's divergence, whose sides the JSON states as
	// the corpus does, and nil for a run that is not flaky.
	divergence *engine.Divergence
}

// detailOf returns the detail of the record of r.
func detailOf(r engine.Result) runDetail {
	return runDetail{result: resultOf(r), divergence: r.Divergence}
}

// MarshalJSON returns the detail as the call record of a property states
// it, in the form of the definition's vectors: the counts as numbers, the
// seed and the token as strings, each drawn value as a typed literal, a
// failure as its failure record, the bounds of a request as an object, and
// null for a field that the outcome does not use.
func (d runDetail) MarshalJSON() ([]byte, error) {
	r := d.result
	out := detailJSON{
		Outcome: r.Outcome, Cases: r.Cases, Rejected: r.Rejected, Seed: strconv.FormatUint(r.Seed, 10),
		Counterexample: r.Counterexample, Failure: r.Failure, Others: r.Others, Coverage: r.Coverage,
	}
	if r.Choices != "" {
		out.Choices = &r.Choices
	}
	if r.Outcome == Skipped {
		out.Reason = &r.Reason
	}
	if v := d.divergence; v != nil {
		out.Divergence = &divergenceJSON{
			What: Difference(v.What), Index: v.Index, Recorded: sideOf(v.Recorded),
			Replayed: sideOf(v.Replayed), Label: r.Divergence.Label, Step: r.Divergence.Step,
		}
	}
	return json.Marshal(out)
}

// detailJSON is the detail of a run as the call record of a property states
// it, in the order that the definition lists its fields.
type detailJSON struct {
	Outcome        Outcome         `json:"outcome"`
	Cases          int             `json:"cases"`
	Rejected       int             `json:"rejected"`
	Seed           string          `json:"seed"`
	Counterexample []Entry         `json:"counterexample"`
	Failure        *assert.Failure `json:"failure"`
	Choices        *string         `json:"choices"`
	Others         []Other         `json:"others"`
	Divergence     *divergenceJSON `json:"divergence"`
	Coverage       *Shortfall      `json:"coverage"`
	Reason         *string         `json:"reason"`
}

// divergenceJSON is a divergence as the call record of a property states
// it: a side is the bounds of a request, the text of a failure's identity,
// a fingerprint, or null, and the label and the step are null where
// [Divergence] states nil.
type divergenceJSON struct {
	What     Difference `json:"what"`
	Index    int        `json:"index"`
	Recorded any        `json:"recorded"`
	Replayed any        `json:"replayed"`
	Label    *string    `json:"label"`
	Step     *Place     `json:"step"`
}

// fields returns the detail as a failure record states it: each field of
// the definition by its name, with its Go value, and nil for a field that
// the outcome does not use. A run that fails did not skip, so its reason is
// nil.
func (d runDetail) fields() map[string]any {
	r := d.result
	detail := map[string]any{
		outcomeField:        r.Outcome,
		casesField:          r.Cases,
		rejectedField:       r.Rejected,
		seedField:           strconv.FormatUint(r.Seed, 10),
		counterexampleField: nil,
		failureField:        nil,
		choicesField:        nil,
		othersField:         nil,
		divergenceField:     nil,
		coverageField:       nil,
		reasonField:         nil,
	}
	if r.Failure != nil {
		detail[counterexampleField], detail[failureField] = r.Counterexample, *r.Failure
	}
	if r.Outcome == Counterexample {
		detail[othersField] = r.Others
	}
	if r.Choices != "" {
		detail[choicesField] = r.Choices
	}
	if r.Divergence != nil {
		detail[divergenceField] = r.Divergence
	}
	if r.Coverage != nil {
		detail[coverageField] = r.Coverage
	}
	return detail
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
