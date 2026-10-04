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

// The parts of the draws vectors that the tests build.
const (
	// countDraw is a draw labelled count of a digit.
	countDraw = `{"label":"count","generator":` + digitGenerator + `}`
	// countOfFour is the value 4 of the draw count.
	countOfFour = `[{"label":"count","value":` + four + `}]`
	// twelve is the typed literal of the integer 12, which no digit is.
	twelve = `{"type":"int","value":12}`
)

// TestDraws checks a draws vector: a body run under a case of Draws
// entries, with the choices of the entries and the value of each draw, or
// the refusal of an entry. Written with testing rather than with this
// library, because a verdict is not written with the subject.
func TestDraws(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		atValue := func(segs ...fault.Segment) fault.Path {
			return inVector(append([]fault.Segment{fault.Field("values"), fault.Index(0)}, segs...)...)
		}
		fourChoices := token.Encode([]choice.Choice{{Kind: choice.Integer, Integer: choice.IntOf(4)}})
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for the choices and the value of an entry that the case takes",
				give: draws(four, `[4]`, countOfFour, null),
			},
			{
				name: "returns nil for the refusal of the value of an entry that the vector states",
				give: draws(twelve, null, null, `{"label":"count","reason":"value"}`),
			},
			{
				name: "returns a fault at the generator of the draw for a generator of an unknown id",
				give: `{"draws":[{"label":"count","generator":` + unknownGenerator + `}],"entries":[]}`,
				wantPath: inVector(
					fault.Field("draws"),
					fault.Index(0),
					fault.Field(generatorAt),
					fault.Field(genAt),
				),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at the value of the entry for an entry that is no typed literal",
				give:       `{"draws":[],"entries":[{"label":"count","value":` + widget + `}]}`,
				wantPath:   inVector(fault.Field("entries"), fault.Index(0), fault.Field(valueAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at error for a refusal of an entry that the run takes",
				give:       draws(four, `[4]`, countOfFour, `{"label":"count","reason":"value"}`),
				wantPath:   inVector(fault.Field(errorAt)),
				wantReason: "the run takes every entry, and the vector states a refusal",
			},
			{
				name:       "returns a fault at error for a refusal of another member than the run's",
				give:       draws(twelve, null, null, `{"label":"count","reason":"label"}`),
				wantPath:   inVector(fault.Field(errorAt)),
				wantReason: `the run refuses another entry, want the label of the entry of a draw labelled "count"`,
			},
			{
				name:       "returns a fault at error for a refusal of another draw's label than the run's",
				give:       draws(twelve, null, null, `{"label":"other","reason":"value"}`),
				wantPath:   inVector(fault.Field(errorAt)),
				wantReason: `the run refuses another entry, want the value of the entry of a draw labelled "other"`,
			},
			{
				name:       "returns a fault at the choices for choices other than the entries' inverses",
				give:       draws(four, `[5]`, countOfFour, null),
				wantPath:   inVector(fault.Field(choicesAt)),
				wantReason: "the choices are " + fourChoices + ", want [5]",
			},
			{
				name:       "returns a fault at the values for another number of values than the case draws",
				give:       draws(four, `[4]`, `[]`, null),
				wantPath:   inVector(fault.Field("values")),
				wantReason: "the case draws 1 values, want 0",
			},
			{
				name:       "returns a fault at the label for a value of another label than its draw's",
				give:       draws(four, `[4]`, `[{"label":"other","value":`+four+`}]`, null),
				wantPath:   atValue(fault.Field("label")),
				wantReason: `the value is of the draw labelled "count", want "other"`,
			},
			{
				name:       "returns a fault at the value for a value that is no typed literal",
				give:       draws(four, `[4]`, `[{"label":"count","value":`+widget+`}]`, null),
				wantPath:   atValue(fault.Field(valueAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at the value for a value other than the one that the case draws",
				give:       draws(four, `[4]`, `[{"label":"count","value":{"type":"int","value":5}}]`, null),
				wantPath:   atValue(fault.Field(valueAt)),
				wantReason: `the draw is int:4, want {"type":"int","value":5}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Draws, tt.give), tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns a fault at error for an entry that the run refuses and the vector does not", func(t *testing.T) {
			t.Parallel()
			err := check(t, conformance.Draws, draws(twelve, null, null, null))
			expectFault(t, err, inVector(fault.Field(errorAt)), "the run refuses an entry, and the vector states none")
			if !errors.Is(err, engine.ErrCannotInvert) {
				t.Fatalf("Check returns %v, want one caused by the refusal of the value", err)
			}
		})
	})
}

// draws returns a draws vector of the draw count of a digit, whose case
// takes the entry count of value, and which states choices, values and the
// refusal.
func draws(value, choices, values, refusal string) string {
	return `{"draws":[` + countDraw + `],"entries":[{"label":"count","value":` + value + `}],` +
		`"choices":` + choices + `,"values":` + values + `,"error":` + refusal + `}`
}
