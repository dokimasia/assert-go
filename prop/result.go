// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// Result is the result of one run of a property: the eleven fields of the
// detail of the run, as Go values. [OnResult] hands it to a function when the
// run ends. The record of the run contains the same values, with the seed as
// decimal text.
//
// A field that the outcome does not use is nil, and a text that the outcome
// does not use is empty:
//
//   - Counterexample and Failure are set for a run with a failing case: a
//     counterexample, and a flaky run whose replay differed.
//   - Others is set for a counterexample, and Choices for a counterexample
//     that is no example of values. An example of values has no replay token.
//   - Divergence is set for a flaky run, and Coverage for a run that missed a
//     coverage requirement.
//   - Reason is set for a skipped run.
type Result struct {
	// Outcome is how the run ended.
	Outcome Outcome
	// Cases is the number of valid cases of the run.
	Cases int
	// Rejected is the number of rejected cases of the run.
	Rejected int
	// Seed is the seed of the run, of the type that [Seed] takes.
	Seed uint64
	// Counterexample is the list of entries of the failing case, in request
	// order. A draw is a [Drawn], and a step of a machine is a [Step].
	Counterexample []Entry
	// Failure is the record of the failing case.
	Failure *assert.Failure
	// Choices is the replay token of the failing case, which [Replay] takes.
	Choices string
	// Others are the other failures of a counterexample.
	Others []Other
	// Divergence is what differed first in a flaky run.
	Divergence *Divergence
	// Coverage is the coverage requirement that the run refuted or left
	// unmet.
	Coverage *Shortfall
	// Reason is the reason of a skipped run.
	Reason string
}

// resultOf returns the result of the engine's run r.
func resultOf(r engine.Result) Result {
	result := Result{
		Outcome: Outcome(r.Outcome), Cases: r.Cases, Rejected: r.Rejected, Seed: r.Seed, Reason: r.Reason,
	}
	if r.Failing != nil {
		failure := failureOf(*r.Failing)
		result.Counterexample, result.Failure = counterexampleOf(*r.Failing, r.Explanation), &failure
	}
	if r.Outcome == engine.Counterexample {
		result.Others = othersOf(r.Others)
		if !r.Failing.Case.Valued() {
			result.Choices = r.Token
		}
	}
	if r.Divergence != nil {
		result.Divergence = divergenceOf(*r.Divergence)
	}
	if r.Shortfall != nil {
		result.Coverage = shortfallOf(*r.Shortfall)
	}
	return result
}
