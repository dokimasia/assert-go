// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// TestToken checks a token vector: the token that choices encode to, or the
// choices that a token decodes to.
func TestToken(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns nil for a vector that states the token of its choices",
				give: `{"choices":[7],"token":"prop1:AAc"}`,
			},
			{
				name: "returns nil for a vector that states the choices of its token",
				give: `{"token":"prop1:AAc","decoded":[7],"error":false}`,
			},
			{
				name: "returns an error for a vector that is no JSON object",
				give: `[]`,
				want: "cannot unmarshal array",
			},
			{
				name: "returns an error that names the index of a choice of no stated kind",
				give: `{"choices":[7,{}],"token":"prop1:AAc"}`,
				want: "choice 1: conformance: {} is no choice",
			},
			{
				name: "returns an error for a token other than the encoding",
				give: `{"choices":[7],"token":"prop1:AAg"}`,
				want: "the token is prop1:AAc, want prop1:AAg",
			},
			{
				name: "returns an error for a refused token that the vector states as decoded",
				give: `{"token":"AAc","decoded":[7],"error":false}`,
				want: "the decoder refuses the token, want [7]: token:",
			},
			{
				name: "returns an error for a decoded token that the vector states as refused",
				give: `{"token":"prop1:AAc","decoded":null,"error":true}`,
				want: "want a refusal",
			},
			{
				name: "returns nil for a refused token that the vector states as refused",
				give: `{"token":"AAc","decoded":null,"error":true}`,
			},
			{
				name: "returns an error for decoded choices of no stated kind",
				give: `{"token":"prop1:AAc","decoded":[{}],"error":false}`,
				want: "{} is no choice",
			},
			{
				name: "returns an error for decoded choices other than the token's",
				give: `{"token":"prop1:AAc","decoded":[8],"error":false}`,
				want: "the token decodes to",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Token, tt.give), tt.want)
			})
		}
	})
}
