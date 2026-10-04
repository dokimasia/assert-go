// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/shape"
)

// checkInverse runs the shape or the generator of an inverse vector
// backwards from its value, and compares the choices, or that no choices
// decode to the value.
func checkInverse(raw json.RawMessage, _ string) error {
	var v struct {
		// Generator is the generator spec, when the vector states one.
		Generator json.RawMessage `json:"generator"`
		// Shape is the shape, when the vector states one.
		Shape json.RawMessage `json:"shape"`
		// Value is the typed literal to run backwards from.
		Value json.RawMessage `json:"value"`
		// Choices are the choices that decode to the value.
		Choices []json.RawMessage `json:"choices"`
		// Error reports whether no choices decode to the value.
		Error bool `json:"error"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	if (v.Generator == nil) == (v.Shape == nil) {
		return fault.New("the vector states a shape or a generator, and not both")
	}
	g, err := generatorOrShape(v.Generator, v.Shape)
	if err != nil {
		return err
	}
	value, err := literal.Decode(v.Value)
	if err != nil {
		return fault.At(err, fault.Field(valueMember))
	}
	choices, refused := engine.Invert(g, value)
	if (refused != nil) != v.Error {
		if refused != nil {
			return fault.At(fault.New("no choices decode to the value, and the vector states choices").Because(refused),
				fault.Field(errorMember))
		}
		return fault.At(fault.New("choices decode to the value, and the vector states none"), fault.Field(errorMember))
	}
	if v.Error {
		return nil
	}
	return at(compareChoices(choices, v.Choices), fault.Field(choicesMember))
}

// generatorOrShape returns the generator of a generator spec, or of a shape
// when spec is nil, and the fault at the member that states it.
func generatorOrShape(spec, shapeText json.RawMessage) (engine.Generator[any], error) {
	if spec != nil {
		g, err := generatorOf(spec)
		if err != nil {
			return engine.Generator[any]{}, fault.At(err, fault.Field(generatorMember))
		}
		return g, nil
	}
	g, err := shape.Read(shapeText)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, fault.Field(shapeMember))
	}
	return g, nil
}
