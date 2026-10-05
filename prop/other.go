// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
)

// Other is a further failure that a run found, with a failure identity of
// its own, shrunk to its smallest case. The shrink budget is shared by
// every failure of a run, so a failure that the budget left unfinished
// states the smallest case found.
type Other struct {
	// Counterexample are the entries of the case in request order: the
	// values it drew, each [Untested] because the explain phase explains the
	// first failure only, and the steps it took.
	Counterexample []Entry
	// Failure is the case's record, as [Case] keeps it.
	Failure assert.Failure
	// Choices is the case's replay token, for [Replay].
	Choices string
}

// otherJSON is another failure as the record of a run states it.
type otherJSON struct {
	Failure        assert.Failure `json:"failure"`
	Counterexample []any          `json:"counterexample"`
	Choices        string         `json:"choices"`
}

// MarshalJSON returns the failure as the record of a run states another
// failure: its failure record, the label and the typed literal of the value
// of each draw, each step as a counterexample states it, and its replay
// token. A value that no typed literal states is an opaque literal.
func (o Other) MarshalJSON() ([]byte, error) {
	entries := make([]any, len(o.Counterexample))
	for i, e := range o.Counterexample {
		entries[i] = e.brief()
	}
	return json.Marshal(otherJSON{Failure: o.Failure, Counterexample: entries, Choices: o.Choices})
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
