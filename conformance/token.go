// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/token"
)

// checkToken encodes the choices of a token vector and compares the token,
// or decodes its token and compares the choices or the refusal.
func checkToken(raw json.RawMessage, _ string) error {
	var v struct {
		// Choices are the choices to encode, and nil for a decoding.
		Choices []json.RawMessage `json:"choices"`
		// Token is the token of the choices, or the token to decode.
		Token string `json:"token"`
		// Decoded are the choices that the token decodes to.
		Decoded []json.RawMessage `json:"decoded"`
		// Error reports whether the decoder refuses the token.
		Error bool `json:"error"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	if v.Choices != nil {
		choices, err := parseChoices(v.Choices)
		if err != nil {
			return fault.At(err, fault.Field(choicesMember))
		}
		if got := token.Encode(choices); got != v.Token {
			return fault.At(fault.New("the token is %s, want %s", got, v.Token), fault.Field(tokenMember))
		}
		return nil
	}
	choices, err := token.Decode(v.Token)
	if v.Error {
		if err == nil {
			return fault.At(fault.New("the token decodes, want a refusal"), fault.Field(errorMember))
		}
		return nil
	}
	if err != nil {
		return fault.At(fault.New("the decoder refuses the token, want %s", jsonOf(v.Decoded)).Because(err),
			fault.Field(tokenMember))
	}
	return at(compareChoices(choices, v.Decoded), fault.Field(decodedMember))
}
