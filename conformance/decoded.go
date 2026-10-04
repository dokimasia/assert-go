// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
)

// engineOutcome is how one case of the engine that decoded a value ended:
// the choices it recorded, and whether it was rejected.
type engineOutcome struct {
	// choices are the recorded choices.
	choices []choice.Choice
	// rejected reports whether the case was rejected.
	rejected bool
}

// decodeWith runs one case that draws a value of g, with the engine's
// function run, and returns the value and how the case ended.
func decodeWith(g engine.Generator[any], run func(engine.Body) engine.Execution) (any, engineOutcome) {
	var got any
	e := run(func(c *engine.Case) { got = engine.Draw(c, g, drawnLabel) })
	return got, engineOutcome{choices: e.Case.Choices(), rejected: e.Status == engine.CaseRejected}
}

// decodedCase is the value that one case decoded, as a vector states it:
// a typed literal, null for a rejected case, and whether the case was
// rejected.
type decodedCase struct {
	// Value is the typed literal of the value.
	Value json.RawMessage `json:"value"`
	// Rejected reports whether the case was rejected.
	Rejected bool `json:"rejected"`
}

// compare returns how the outcome of e, which drew got, differs from the
// one that d states with the choices recorded, or nil when they match. The
// vector states recorded at its member member. A fault states the choices
// of the case as their replay token.
//
// The value matches when got has the canonical text of the typed literal
// that d states: every integer type is one type and every float type
// another, a float compares by its bits, and a map compares by its entries
// in any order.
func (d decodedCase) compare(e engineOutcome, got any, recorded []json.RawMessage, member string) error {
	if e.rejected != d.Rejected {
		return fault.At(fault.New("the rejection is %t, want %t", e.rejected, d.Rejected), fault.Field(rejectedMember))
	}
	same, err := sameChoices(e.choices, recorded)
	if err != nil {
		return fault.At(err, fault.Field(member))
	}
	if !same {
		return fault.At(fault.New("the case records %s, want %s", token.Encode(e.choices), jsonOf(recorded)),
			fault.Field(member))
	}
	if d.Rejected {
		return nil
	}
	same, err = sameValue(got, d.Value)
	if err != nil {
		return fault.At(err, fault.Field(valueMember))
	}
	if !same {
		return fault.At(
			fault.New("the value is %s, want %s", literal.Canonical(got), d.Value),
			fault.Field(valueMember),
		)
	}
	return nil
}
