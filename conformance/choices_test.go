// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// TestChoices checks the corpus form of a choice, through the token that a
// token vector encodes its choices to, and the comparison of a case's
// choices with the ones that a decoding vector states.
func TestChoices(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			kind conformance.VectorKind
			give string
			want string
		}{
			{
				name: "returns an error for a choice of no stated kind",
				kind: conformance.Token,
				give: `{"choices":[{"integer":7}],"token":"prop1:AAc"}`,
				want: `{"integer":7} is no choice`,
			},
			{
				name: "returns an error for a float choice of an unknown name",
				kind: conformance.Token,
				give: `{"choices":[{"float":"Huge"}],"token":"prop1:"}`,
				want: `unrecognized float literal "Huge"`,
			},
			{
				name: "returns an error for an integer choice within 2^53 - 1 stated as a decimal string",
				kind: conformance.Token,
				give: `{"choices":["7"],"token":"prop1:AAc"}`,
				want: "7 is within 2^53 - 1",
			},
			{
				name: "returns an error for an integer choice of a fraction",
				kind: conformance.Token,
				give: `{"choices":[1.5],"token":"prop1:"}`,
				want: "decode scalar",
			},
			{
				name: "returns an error for recorded choices of another length",
				kind: conformance.Decoding,
				give: decoded(digitGenerator, `[7]`, `[7,0]`, `{"type":"int","value":7}`),
				want: "the case records",
			},
			{
				name: "returns an error for a recorded integer of another value",
				kind: conformance.Decoding,
				give: decoded(digitGenerator, `[7]`, `[6]`, `{"type":"int","value":7}`),
				want: "the case records",
			},
			{
				name: "returns an error for a recorded choice of another kind",
				kind: conformance.Decoding,
				give: decoded(digitGenerator, `[7]`, `[{"float":7}]`, `{"type":"int","value":7}`),
				want: "the case records",
			},
			{
				name: "returns an error for a recorded float of the other sign of zero",
				kind: conformance.Decoding,
				give: decoded(`{"gen":"float","min":-1,"max":1}`, `[{"float":0.0}]`, `[{"float":-0.0}]`,
					`{"type":"float","value":0.0}`),
				want: "the case records",
			},
			{
				name: "returns an error for a recorded choice of no stated kind",
				kind: conformance.Decoding,
				give: decoded(digitGenerator, `[7]`, `[{"integer":7}]`, `{"type":"int","value":7}`),
				want: `{"integer":7} is no choice`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, tt.kind, tt.give), tt.want)
			})
		}
	})
}
