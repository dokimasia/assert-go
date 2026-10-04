// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/shape"
)

// seededVector is a vector that decodes a generator or a shape from the
// first cases of a seed, and states each case's choices and value.
type seededVector struct {
	// Generator is the generator spec of a generation vector.
	Generator json.RawMessage `json:"generator"`
	// Shape is the shape of a shapes vector.
	Shape json.RawMessage `json:"shape"`
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

// checkGeneration decodes the generator of a generation vector from the
// first cases of its seed, and compares each case's choices and value.
func checkGeneration(raw json.RawMessage, _ string) error {
	var v seededVector
	if err := decode(raw, &v); err != nil {
		return err
	}
	g, err := generatorOf(v.Generator)
	if err != nil {
		return fault.At(err, fault.Field(generatorMember))
	}
	return v.check(g)
}

// checkShapes reads the shape of a shapes vector, decodes it from the first
// cases of its seed, and compares each case's choices and value.
func checkShapes(raw json.RawMessage, _ string) error {
	var v seededVector
	if err := decode(raw, &v); err != nil {
		return err
	}
	g, err := shape.Read(v.Shape)
	if err != nil {
		return fault.At(err, fault.Field(shapeMember))
	}
	return v.check(g)
}

// check decodes g from the first cases of the vector's seed, and compares
// each case's choices and value.
func (v seededVector) check(g engine.Generator[any]) error {
	seed, err := parseSeed(v.Seed)
	if err != nil {
		return err
	}
	if uint64(len(v.Cases)) != v.Count {
		return fault.At(fault.New("the vector states %d cases of %d", len(v.Cases), v.Count), fault.Field(casesMember))
	}
	for i, want := range v.Cases {
		got, outcome := decodeWith(g, func(body engine.Body) engine.Execution {
			return engine.Generate(body, seed, uint64(i), nil)
		})
		if err := want.compare(outcome, got, want.Choices, choicesMember); err != nil {
			return fault.At(err, fault.Field(casesMember), fault.Index(i))
		}
	}
	return nil
}
