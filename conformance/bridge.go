// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/hex"
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// checkBridge decodes the generator of a bridge vector from its fuzzer
// bytes, and compares the choices, the value and the rejection.
func checkBridge(raw json.RawMessage, _ string) error {
	var v struct {
		decodedCase
		// Generator is the generator spec.
		Generator json.RawMessage `json:"generator"`
		// Bytes are the fuzzer's bytes in hexadecimal.
		Bytes string `json:"bytes"`
		// Choices are the choices the case records.
		Choices []json.RawMessage `json:"choices"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	g, err := generatorOf(v.Generator)
	if err != nil {
		return fault.At(err, fault.Field(generatorMember))
	}
	data, err := hex.DecodeString(v.Bytes)
	if err != nil {
		return fault.At(fault.New("the bytes are no hexadecimal").Because(err), fault.Field(bytesMember))
	}
	bridged := func(body engine.Body) engine.Execution { return engine.Bridge(body, data, engine.Settings{}) }
	got, outcome := decodeWith(g, bridged)
	return v.compare(outcome, got, v.Choices, choicesMember)
}
