// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
)

// boolShape is the shape of a boolean.
const boolShape = `{"shape":"bool"}`

// TestInverse checks an inverse vector: a shape or a generator run
// backwards from a value to its choices, or to no choices. Written with
// testing rather than with this library, because a verdict is not written
// with the subject.
func TestInverse(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		const shapeOrGenerator = "the vector states a shape or a generator, and not both"
		trueChoices := token.Encode([]choice.Choice{{Kind: choice.Integer, Integer: choice.IntOf(1)}})
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for the choices of a present value of an optional generator",
				give: `{"generator":{"gen":"optional","of":` + digitGenerator + `},"value":` + four +
					`,"choices":[1,4],"error":false}`,
			},
			{
				name: "returns nil for a refusal that the vector states of a value that no choices decode to",
				give: `{"shape":` + boolShape + `,"value":` + four + `,"choices":null,"error":true}`,
			},
			{
				name:       "returns a fault for an inverse vector that states neither a shape nor a generator",
				give:       `{"value":` + four + `,"choices":[4]}`,
				wantPath:   inVector(),
				wantReason: shapeOrGenerator,
			},
			{
				name:       "returns a fault for an inverse vector that states both a shape and a generator",
				give:       `{"shape":` + boolShape + `,"generator":{"gen":"boolean"},"value":` + four + `,"choices":[4]}`,
				wantPath:   inVector(),
				wantReason: shapeOrGenerator,
			},
			{
				name:       "returns a fault at the shape for a shape of no shape of the vocabulary",
				give:       `{"shape":{"shape":"widget"},"value":` + four + `,"choices":[4]}`,
				wantPath:   inVector(fault.Field(shapeAt), fault.Field(shapeAt)),
				wantReason: `"widget" is no shape of the vocabulary`,
			},
			{
				name:       "returns a fault at the generator for a generator of an unknown id",
				give:       `{"generator":` + unknownGenerator + `,"value":` + four + `,"choices":[4]}`,
				wantPath:   inVector(fault.Field(generatorAt), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at the value for a value that is no typed literal",
				give:       `{"shape":` + boolShape + `,"value":` + widget + `,"choices":[1]}`,
				wantPath:   inVector(fault.Field(valueAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at error for a refusal that the vector states of a value that choices decode to",
				give:       `{"shape":` + boolShape + `,"value":{"type":"bool","value":true},"choices":null,"error":true}`,
				wantPath:   inVector(fault.Field(errorAt)),
				wantReason: "choices decode to the value, and the vector states none",
			},
			{
				name:       "returns a fault at the choices for choices other than the ones that decode to the value",
				give:       `{"shape":` + boolShape + `,"value":{"type":"bool","value":true},"choices":[0],"error":false}`,
				wantPath:   inVector(fault.Field(choicesAt)),
				wantReason: "the choices are " + trueChoices + ", want [0]",
			},
			{
				name:       "returns a fault at the choice for a choice of no corpus form",
				give:       `{"shape":` + boolShape + `,"value":{"type":"bool","value":true},"choices":["x"],"error":false}`,
				wantPath:   inVector(fault.Field(choicesAt), fault.Index(0)),
				wantReason: `"x" is no canonical integer of 64 bits`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Inverse, tt.give), tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns a fault at error for choices that the vector states of a value that no choices decode to",
			func(t *testing.T) {
				t.Parallel()
				err := check(t, conformance.Inverse,
					`{"shape":`+boolShape+`,"value":`+four+`,"choices":[4],"error":false}`)
				expectFault(t, err, inVector(fault.Field(errorAt)),
					"no choices decode to the value, and the vector states choices")
				if !errors.Is(err, engine.ErrCannotInvert) {
					t.Fatalf("Check returns %v, want one caused by the refusal of the inverse", err)
				}
			})
	})
}
