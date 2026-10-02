// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"

	"go.dokimi.dev/assert/internal/prop/token"
)

// checkToken encodes the choices of a token vector and compares the token,
// or decodes its token and compares the choices or the refusal.
func checkToken(raw json.RawMessage) error {
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
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	if v.Choices != nil {
		choices, err := parseChoices(v.Choices)
		if err != nil {
			return err
		}
		if got := token.Encode(choices); got != v.Token {
			return fmt.Errorf("the token is %s, want %s", got, v.Token)
		}
		return nil
	}
	choices, err := token.Decode(v.Token)
	if v.Error {
		if err == nil {
			return fmt.Errorf("the token decodes to %v, want a refusal", choices)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("the decoder refuses the token, want %s: %w", jsonOf(v.Decoded), err)
	}
	same, err := sameChoices(choices, v.Decoded)
	if err != nil {
		return err
	}
	if !same {
		return fmt.Errorf("the token decodes to %v, want %s", choices, jsonOf(v.Decoded))
	}
	return nil
}
