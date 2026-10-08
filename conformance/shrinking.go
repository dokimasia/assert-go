// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// failsWhen is the identity of the failure of a shrinking vector's body.
const failsWhen = "fails-when"

// checkShrinking runs the one-draw property of a shrinking vector, which
// fails when its predicate is true, and compares the outcome, the valid
// cases, the minimal value, its choices and its token, and the runs that
// shrinking and explaining spent.
func checkShrinking(raw json.RawMessage, _ string) error {
	var v struct {
		// Generator is the generator spec of the draw.
		Generator json.RawMessage `json:"generator"`
		// FailsWhen is the predicate of the failure.
		FailsWhen json.RawMessage `json:"fails-when"`
		// Seed is the seed in decimal.
		Seed string `json:"seed"`
		// Budget is the shrink budget, the default when nil.
		Budget *int `json:"budget"`
		// Outcome is how the run ends.
		Outcome string `json:"outcome"`
		// Cases is the number of valid cases.
		Cases int `json:"cases"`
		// Value is the minimal value's typed literal, null for a pass.
		Value json.RawMessage `json:"value"`
		// Choices are the minimal case's choices, null for a pass.
		Choices []json.RawMessage `json:"choices"`
		// Token is the minimal case's token, null for a pass.
		Token *string `json:"token"`
		// Runs is the number of runs that shrinking and explaining spent.
		Runs int `json:"runs"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	g, err := generatorOf(v.Generator)
	if err != nil {
		return fault.At(err, fault.Field(generatorMember))
	}
	fails, err := predicateOf(v.FailsWhen)
	if err != nil {
		return fault.At(err, fault.Field(failsWhenMember))
	}
	seed, err := parseSeed(v.Seed)
	if err != nil {
		return err
	}
	s := engine.Settings{
		Seed:       seed,
		Cases:      engine.DefaultCases,
		MaxChoices: engine.MaxChoices,
		Shrink:     engine.DefaultShrink,
		Explain:    true,
		Workers:    1,
	}
	if v.Budget != nil {
		s.Shrink = *v.Budget
	}
	draw := prop.Generator[any](g)
	r := engine.Run(engineBody(func(c *prop.Case) {
		if fails(c.Draw(draw, drawnLabel)) {
			fail(c, failsWhen)
		}
	}), s)
	if r.Outcome.String() != v.Outcome {
		return fault.At(fault.New("the run ends as %v, want %s", r.Outcome, v.Outcome), fault.Field(outcomeMember))
	}
	if r.Cases != v.Cases {
		return fault.At(fault.New("the run has %d valid cases, want %d", r.Cases, v.Cases), fault.Field(casesMember))
	}
	if r.Runs != v.Runs {
		return fault.At(fault.New("shrinking and explaining spend %d runs, want %d", r.Runs, v.Runs),
			fault.Field(runsMember))
	}
	if r.Failing == nil {
		if v.Token != nil {
			return fault.At(fault.New("the run has no minimal case, want the token %s", *v.Token),
				fault.Field(tokenMember))
		}
		if string(v.Value) != jsonNull {
			return fault.At(fault.New("the run has no minimal case, want %s", v.Value), fault.Field(valueMember))
		}
		return nil
	}
	if v.Token == nil || r.Token != *v.Token {
		return fault.At(fault.New("the token is %s, want %s", r.Token, jsonOf(v.Token)), fault.Field(tokenMember))
	}
	minimal := decodedCase{Value: v.Value}
	outcome := engineOutcome{choices: r.Failing.Case.Choices()}
	return minimal.compare(outcome, r.Failing.Case.Draws()[0].Value, v.Choices, choicesMember)
}
