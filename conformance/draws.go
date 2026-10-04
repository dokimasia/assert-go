// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"errors"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// labelled is a label and a typed literal, as a draws vector states an
// entry and a value.
type labelled struct {
	// Label is the label of the draw.
	Label string `json:"label"`
	// Value is the typed literal of the value.
	Value json.RawMessage `json:"value"`
}

// drawsVector is a draws vector: the draws of a body, the entries of its
// case of Draws, and what the case states, or the refusal.
type drawsVector struct {
	// Draws are the draws of the body, each a label and a generator spec,
	// in order.
	Draws []struct {
		// Label is the label of the draw.
		Label string `json:"label"`
		// Generator is the generator spec.
		Generator json.RawMessage `json:"generator"`
	} `json:"draws"`
	// Entries are the entries of the case, in order.
	Entries []labelled `json:"entries"`
	// Choices are the choices that decode to the entries' values.
	Choices []json.RawMessage `json:"choices"`
	// Values are the label and the value of each draw of the case.
	Values []labelled `json:"values"`
	// Error is the refusal of an entry, and nil for a case that takes each.
	Error *struct {
		// Label is the label of the draw that refused its entry.
		Label string `json:"label"`
		// Reason is why: label or value, the member of the entry that the
		// draw refused.
		Reason string `json:"reason"`
	} `json:"error"`
}

// checkDraws runs a body of the draws of a draws vector under a case of its
// entries, and compares the choices of the entries and the value of each
// draw, or the refusal. The choices are each entry's inverse under its
// draw's generator.
func checkDraws(raw json.RawMessage, _ string) error {
	var v drawsVector
	if err := decode(raw, &v); err != nil {
		return err
	}
	gens := make([]engine.Generator[any], len(v.Draws))
	for i, d := range v.Draws {
		g, err := generatorOf(d.Generator)
		if err != nil {
			return fault.At(err, fault.Field(drawsMember), fault.Index(i), fault.Field(generatorMember))
		}
		gens[i] = g
	}
	entries := make([]engine.Entry, len(v.Entries))
	for i, e := range v.Entries {
		value, err := literal.Decode(e.Value)
		if err != nil {
			return fault.At(err, fault.Field(entriesMember), fault.Index(i), fault.Field(valueMember))
		}
		entries[i] = engine.Entry{Label: e.Label, Value: value}
	}
	var drawn []any
	body := func(c *engine.Case) {
		values := make([]any, len(gens))
		for i, g := range gens {
			values[i] = engine.Draw(c, g, v.Draws[i].Label)
		}
		if drawn == nil {
			drawn = values
		}
	}
	r := engine.Run(body, engine.Settings{Cases: 1, MaxChoices: engine.MaxChoices, Draws: entries})
	if v.Error != nil || r.Refused != nil {
		return v.compareRefusal(r.Refused)
	}
	return v.compare(gens, entries, drawn)
}

// compareRefusal returns how the refusal of a run differs from the one that
// v states, or nil when they match. The refusal of the engine is a fault
// whose path starts at the index of the refused entry and the member that
// the draw refused, so it matches when that path starts at the entry of a
// draw of the vector's label and at the member that the vector's reason
// names.
func (v drawsVector) compareRefusal(refused error) error {
	if v.Error == nil {
		return fault.At(fault.New("the run refuses an entry, and the vector states none").Because(refused),
			fault.Field(errorMember))
	}
	if refused == nil {
		return fault.At(fault.New("the run takes every entry, and the vector states a refusal"),
			fault.Field(errorMember))
	}
	f, ok := errors.AsType[*fault.Error](refused)
	for n, d := range v.Draws {
		at := fault.Path{fault.Index(n), fault.Field(v.Error.Reason)}
		if ok && d.Label == v.Error.Label && slices.Equal(f.Path[:min(len(at), len(f.Path))], at) {
			return nil
		}
	}
	return fault.At(fault.New("the run refuses another entry, want the %s of the entry of a draw labelled %q",
		v.Error.Reason, v.Error.Label).Because(refused), fault.Field(errorMember))
}

// compare returns how a case of Draws that drew drawn differs from what v
// states, or nil when they match. The choices that v states are each
// entry's inverse under its draw's generator, in order.
func (v drawsVector) compare(gens []engine.Generator[any], entries []engine.Entry, drawn []any) error {
	var computed []choice.Choice
	for i := range min(len(gens), len(entries)) {
		// The case took each entry, so each runs back to choices.
		choices, _ := engine.Invert(gens[i], entries[i].Value)
		computed = append(computed, choices...)
	}
	if err := compareChoices(computed, v.Choices); err != nil {
		return fault.At(err, fault.Field(choicesMember))
	}
	if len(drawn) != len(v.Values) {
		return fault.At(fault.New("the case draws %d values, want %d", len(drawn), len(v.Values)),
			fault.Field(valuesMember))
	}
	for i, want := range v.Values {
		if want.Label != v.Draws[i].Label {
			return fault.At(fault.New("the value is of the draw labelled %q, want %q", v.Draws[i].Label, want.Label),
				fault.Field(valuesMember), fault.Index(i), fault.Field(labelMember))
		}
		same, err := sameValue(drawn[i], want.Value)
		if err != nil {
			return fault.At(err, fault.Field(valuesMember), fault.Index(i), fault.Field(valueMember))
		}
		if !same {
			return fault.At(fault.New("the draw is %s, want %s", literal.Canonical(drawn[i]), want.Value),
				fault.Field(valuesMember), fault.Index(i), fault.Field(valueMember))
		}
	}
	return nil
}
