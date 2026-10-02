// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"strconv"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// checkGeneration decodes the generator of a generation vector from the
// first cases of its seed, and compares each case's choices and value.
func checkGeneration(raw json.RawMessage) error {
	var v struct {
		// Generator is the generator spec.
		Generator json.RawMessage `json:"generator"`
		// Seed is the seed in decimal.
		Seed string `json:"seed"`
		// Count is the number of cases.
		Count uint64 `json:"count"`
		// Cases are the cases in order.
		Cases []struct {
			decodedCase
			// Choices are the choices the case records.
			Choices []json.RawMessage `json:"choices"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	g, err := generatorOf(v.Generator)
	if err != nil {
		return err
	}
	seed, err := strconv.ParseUint(v.Seed, 10, 64)
	if err != nil {
		return err
	}
	if uint64(len(v.Cases)) != v.Count {
		return fmt.Errorf("the vector states %d cases of %d", len(v.Cases), v.Count)
	}
	for i, want := range v.Cases {
		got, outcome := decodeWith(g, func(body engine.Body) engine.Execution {
			return engine.Generate(body, seed, uint64(i), nil)
		})
		if err := want.compare(outcome, got, want.Choices); err != nil {
			return fmt.Errorf("case %d: %w", i, err)
		}
	}
	return nil
}
