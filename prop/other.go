// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/literal"
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

// otherJSON is another failure as the record of a run states it.
type otherJSON struct {
	Failure        assert.Failure `json:"failure"`
	Counterexample []labelled     `json:"counterexample"`
	Choices        string         `json:"choices"`
}

// labelled is a draw of another failure's case as the record of a run
// states it.
type labelled struct {
	Label string          `json:"label"`
	Value json.RawMessage `json:"value"`
}

// MarshalJSON returns the failure as the record of a run states another
// failure: its failure record, the label and the typed literal of the value
// of each draw, and its replay token. A value that no typed literal states
// is an opaque literal.
func (o Other) MarshalJSON() ([]byte, error) {
	draws := make([]labelled, len(o.Counterexample))
	for i, d := range o.Counterexample {
		draws[i] = labelled{Label: d.Label, Value: literal.Detail(d.Value)}
	}
	return json.Marshal(otherJSON{Failure: o.Failure, Counterexample: draws, Choices: o.Choices})
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
