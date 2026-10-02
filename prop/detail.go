// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"strconv"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The assertion of a failing run's record, and the names of its detail
// fields in the order the definition lists them.
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

// detailOf returns the ten detail fields of the record of a run that did
// not pass. A field that the outcome does not use is nil:
//
//   - counterexample and failure are set for a run with a failing case: a
//     counterexample, and a flaky run whose replay differed.
//   - choices and others are set for a counterexample.
//   - divergence is set for a flaky run, and coverage for a run that missed
//     a coverage requirement.
func detailOf(r engine.Result) map[string]any {
	detail := map[string]any{
		outcomeField:        Outcome(r.Outcome),
		casesField:          r.Cases,
		rejectedField:       r.Rejected,
		seedField:           strconv.FormatUint(r.Seed, 10),
		counterexampleField: nil,
		failureField:        nil,
		choicesField:        nil,
		othersField:         nil,
		divergenceField:     nil,
		coverageField:       nil,
	}
	if r.Failing != nil {
		detail[counterexampleField] = counterexampleOf(*r.Failing, r.Explanation)
		detail[failureField] = failureOf(*r.Failing)
	}
	if r.Outcome == engine.Counterexample {
		detail[choicesField] = r.Token
		detail[othersField] = othersOf(r.Others)
	}
	if r.Divergence != nil {
		detail[divergenceField] = divergenceOf(*r.Divergence)
	}
	if r.Shortfall != nil {
		detail[coverageField] = shortfallOf(*r.Shortfall)
	}
	return detail
}

// failureOf returns the record of a failed case: the first record that its
// body kept, or for a panic a record without an assertion whose contract is
// the panic's value, with the value's type and the stack in its detail.
func failureOf(e engine.Execution) assert.Failure {
	if e.Panic == nil {
		return e.Case.Failures()[0]
	}
	return assert.Failure{
		Contract: fmt.Sprint(e.Panic),
		Detail:   map[string]any{panicField: e.Identity.Panic, stackField: string(e.Stack)},
		Where:    e.Identity.Where,
	}
}
