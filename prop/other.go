// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
)

// Other is a further failure that a run found, with a failure identity of
// its own, shrunk to its smallest case. The shrink budget is shared by
// every failure of a run, so a failure that the budget left unfinished
// states the smallest case found.
type Other struct {
	// Counterexample are the values the case drew, in order, each
	// [Untested]: the explain phase explains the first failure only.
	Counterexample []Drawn
	// Failure is the case's record, as [Case] keeps it.
	Failure assert.Failure
	// Choices is the case's replay token, for [Replay].
	Choices string
}

// othersOf returns the other failures of a run in the order the run found
// them, and an empty list for none.
func othersOf(executions []engine.Execution) []Other {
	others := make([]Other, 0, len(executions))
	for _, e := range executions {
		others = append(others, Other{
			Counterexample: counterexampleOf(e, nil),
			Failure:        failureOf(e),
			Choices:        token.Encode(e.Case.Choices()),
		})
	}
	return others
}
