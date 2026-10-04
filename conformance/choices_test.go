// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/token"
)

// TestChoices checks the corpus form of a choice, through the token that a
// token vector encodes its choices to, and the comparison of a case's
// choices with the ones that a decoding vector states, by kind and by value,
// a float by its bits. Written with testing rather than with this library,
// because a verdict is not written with the subject.
func TestChoices(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		atFirst := inVector(fault.Field(choicesAt), fault.Index(0))
		atRecorded := inVector(fault.Field(recordedAt))
		tests := []struct {
			name       string
			kind       conformance.VectorKind
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault at the choice for a choice of no stated kind",
				kind:       conformance.Token,
				give:       `{"choices":[{"integer":7}],"token":"prop1:AAc"}`,
				wantPath:   atFirst,
				wantReason: `{"integer":7} is no choice`,
			},
			{
				name:       "returns a fault at the float of a float choice of an unknown name",
				kind:       conformance.Token,
				give:       `{"choices":[{"float":"Huge"}],"token":"prop1:"}`,
				wantPath:   inVector(fault.Field(choicesAt), fault.Index(0), fault.Field("float")),
				wantReason: `"Huge" is none of the names NaN, Inf and -Inf`,
			},
			{
				name:       "returns a fault at the choice for an integer within 2^53 - 1 stated as a decimal string",
				kind:       conformance.Token,
				give:       `{"choices":["7"],"token":"prop1:AAc"}`,
				wantPath:   atFirst,
				wantReason: "7 is within 2^53 - 1, which a JSON number states",
			},
			{
				name:       "returns a fault at the choice for an integer choice of a fraction",
				kind:       conformance.Token,
				give:       `{"choices":[1.5],"token":"prop1:"}`,
				wantPath:   atFirst,
				wantReason: "the value is no int",
			},
			{
				name:       "returns a fault for recorded choices of another length",
				kind:       conformance.Decoding,
				give:       decoded(digitGenerator, `[7]`, `[7,0]`, `{"type":"int","value":7}`),
				wantPath:   atRecorded,
				wantReason: "the case records prop1:AAc, want [7,0]",
			},
			{
				name:       "returns a fault for a recorded integer of another value",
				kind:       conformance.Decoding,
				give:       decoded(digitGenerator, `[7]`, `[6]`, `{"type":"int","value":7}`),
				wantPath:   atRecorded,
				wantReason: "the case records prop1:AAc, want [6]",
			},
			{
				name:       "returns a fault for a recorded choice of another kind",
				kind:       conformance.Decoding,
				give:       decoded(digitGenerator, `[7]`, `[{"float":7}]`, `{"type":"int","value":7}`),
				wantPath:   atRecorded,
				wantReason: `the case records prop1:AAc, want [{"float":7}]`,
			},
			{
				name: "returns a fault for a recorded float of the other sign of zero",
				kind: conformance.Decoding,
				give: decoded(`{"gen":"float","min":-1,"max":1}`, `[{"float":0.0}]`, `[{"float":-0.0}]`,
					`{"type":"float","value":0.0}`),
				wantPath: atRecorded,
				wantReason: "the case records " + token.Encode([]choice.Choice{{Kind: choice.Float}}) +
					`, want [{"float":-0.0}]`,
			},
			{
				name:       "returns a fault at the choice for a recorded choice of no stated kind",
				kind:       conformance.Decoding,
				give:       decoded(digitGenerator, `[7]`, `[{"integer":7}]`, `{"type":"int","value":7}`),
				wantPath:   inVector(fault.Field(recordedAt), fault.Index(0)),
				wantReason: `{"integer":7} is no choice`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, tt.kind, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}
