// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Relevance -linecomment -output=drawn.string_gen.go

// Relevance is what the explain phase found for one draw of a
// counterexample. Each value converts from the engine's relevance of the
// same spelling.
type Relevance uint8

const (
	// Untested is a draw that the explain phase did not test: explaining is
	// off, the draw makes no choice, none of its fillings decodes a value, or
	// the budget ran out first.
	Untested Relevance = 0 // untested
	// AnyValueFails is a draw whose every filling that decodes a value fails
	// the same way, so the drawn value takes no part in the failure.
	AnyValueFails Relevance = 1 // any-value-fails
	// ValueMatters is a draw with a filling that passes or fails another
	// way, so the drawn value takes part in the failure.
	ValueMatters Relevance = 2 // value-matters
)

// Valid reports whether r is one of the three relevances.
func (r Relevance) Valid() bool {
	return r <= ValueMatters
}

// Drawn is one value of a counterexample, in the order the body drew it.
// The zero Drawn is an untested draw of nil under the empty label.
type Drawn struct {
	// Label is the label the body drew the value under.
	Label string
	// Value is the value the generator decoded, of the generator's type.
	Value any
	// Relevance is what the explain phase found for the draw.
	Relevance Relevance
	// NearestPassing is the value one step towards the target that passes,
	// for an integer or a duration whose value matters, or a value that
	// [Generator.Map] maps from one, of the generator's type. It is nil for
	// every other draw.
	NearestPassing any
}

// drawnJSON is a draw of a counterexample as the record of a run states it.
type drawnJSON struct {
	Label          string          `json:"label"`
	Value          json.RawMessage `json:"value"`
	AnyValueFails  *bool           `json:"any-value-fails"`
	NearestPassing json.RawMessage `json:"nearest-passing"`
}

// MarshalJSON returns the draw as the record of a run states a draw of its
// counterexample: the label, the typed literal of the value, whether any
// value fails, which is null for a draw that the explain phase did not
// test, and the typed literal of the nearest passing value, which is null
// for none. A value that no typed literal states is an opaque literal.
func (d Drawn) MarshalJSON() ([]byte, error) {
	out := drawnJSON{Label: d.Label, Value: literal.Detail(d.Value)}
	if d.Relevance != Untested {
		fails := d.Relevance == AnyValueFails
		out.AnyValueFails = &fails
	}
	if d.NearestPassing != nil {
		out.NearestPassing = literal.Detail(d.NearestPassing)
	}
	return json.Marshal(out)
}

// counterexampleOf returns the draws of e's case in order, each with the
// explanation at its index. explained is the run's explanation of e, which
// states one entry for each draw, or nil for a case that the run did not
// explain.
func counterexampleOf(e engine.Execution, explained []engine.Explained) []Drawn {
	draws := e.Case.Draws()
	out := make([]Drawn, len(draws))
	for i, d := range draws {
		out[i] = Drawn{Label: d.Label, Value: d.Value}
		if i < len(explained) {
			out[i].Relevance, out[i].NearestPassing = Relevance(explained[i].Relevance), explained[i].NearestPassing
		}
	}
	return out
}
