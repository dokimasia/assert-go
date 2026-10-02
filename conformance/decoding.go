// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// engineOutcome is how one case of the engine that decoded a value ended:
// the choices it recorded, and whether it was rejected.
type engineOutcome struct {
	// choices are the recorded choices.
	choices []choice.Choice
	// rejected reports whether the case was rejected.
	rejected bool
}

// decodeWith runs one case that draws a value of g, with the engine's
// function run, and returns the value and how the case ended.
func decodeWith(g engine.Generator[any], run func(engine.Body) engine.Execution) (any, engineOutcome) {
	var got any
	e := run(func(c *engine.Case) { got = engine.Draw(c, g, drawnLabel) })
	return got, engineOutcome{choices: e.Case.Choices(), rejected: e.Status == engine.CaseRejected}
}

// checkDecoding replays the choices of a decoding vector into its
// generator, and compares the recorded choices, the value and the
// rejection.
func checkDecoding(raw json.RawMessage) error {
	var v struct {
		decodedCase
		// Generator is the generator spec.
		Generator json.RawMessage `json:"generator"`
		// Choices are the choices to replay.
		Choices []json.RawMessage `json:"choices"`
		// Recorded are the choices the case records.
		Recorded []json.RawMessage `json:"recorded"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	g, err := generatorOf(v.Generator)
	if err != nil {
		return err
	}
	choices, err := parseChoices(v.Choices)
	if err != nil {
		return err
	}
	got, outcome := decodeWith(g, func(body engine.Body) engine.Execution { return engine.Replay(body, choices, nil) })
	return v.compare(outcome, got, v.Recorded)
}
