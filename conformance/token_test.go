// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/token"
)

// TestToken checks a token vector: the token that choices encode to, or the
// choices that a token decodes to. Written with testing rather than with
// this library, because a verdict is not written with the subject.
func TestToken(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
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
				name: "returns nil for a refused token that the vector states as refused",
				give: `{"token":"AAc","decoded":null,"error":true}`,
			},
			{
				name:       "returns a fault at the index of a choice of no stated kind",
				give:       `{"choices":[7,{}],"token":"prop1:AAc"}`,
				wantPath:   inVector(fault.Field(choicesAt), fault.Index(1)),
				wantReason: "{} is no choice",
			},
			{
				name:       "returns a fault at the token for a token other than the encoding",
				give:       `{"choices":[7],"token":"prop1:AAg"}`,
				wantPath:   inVector(fault.Field("token")),
				wantReason: "the token is prop1:AAc, want prop1:AAg",
			},
			{
				name:       "returns a fault at error for a decoded token that the vector states as refused",
				give:       `{"token":"prop1:AAc","decoded":null,"error":true}`,
				wantPath:   inVector(fault.Field(errorAt)),
				wantReason: "the token decodes, want a refusal",
			},
			{
				name:       "returns a fault at the index of a decoded choice of no stated kind",
				give:       `{"token":"prop1:AAc","decoded":[{}],"error":false}`,
				wantPath:   inVector(fault.Field("decoded"), fault.Index(0)),
				wantReason: "{} is no choice",
			},
			{
				name:       "returns a fault at decoded for decoded choices other than the token's",
				give:       `{"token":"prop1:AAc","decoded":[8],"error":false}`,
				wantPath:   inVector(fault.Field("decoded")),
				wantReason: "the choices are prop1:AAc, want [8]",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Token, tt.give), tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns a fault at the token for a refused token that the vector states as decoded", func(t *testing.T) {
			t.Parallel()
			err := check(t, conformance.Token, `{"token":"AAc","decoded":[7],"error":false}`)
			expectFault(t, err, inVector(fault.Field("token")), "the decoder refuses the token, want [7]")
			if !errors.Is(err, token.ErrInvalid) {
				t.Fatalf("Check returns %v, want one caused by the decoder's refusal", err)
			}
		})
	})
}
