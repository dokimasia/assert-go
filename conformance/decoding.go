// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// checkDecoding replays the choices of a decoding vector into its
// generator, and compares the recorded choices, the value and the
// rejection.
func checkDecoding(raw json.RawMessage, _ string) error {
	var v struct {
		decodedCase
		// Generator is the generator spec.
		Generator json.RawMessage `json:"generator"`
		// Choices are the choices to replay.
		Choices []json.RawMessage `json:"choices"`
		// Recorded are the choices the case records.
		Recorded []json.RawMessage `json:"recorded"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	g, err := generatorOf(v.Generator)
	if err != nil {
		return fault.At(err, fault.Field(generatorMember))
	}
	choices, err := parseChoices(v.Choices)
	if err != nil {
		return fault.At(err, fault.Field(choicesMember))
	}
	got, outcome := decodeWith(g, func(body engine.Body) engine.Execution { return engine.Replay(body, choices, nil) })
	return v.compare(outcome, got, v.Recorded, recordedMember)
}
