// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"strconv"

	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// failsWhen is the identity of the failure of a shrinking vector's body.
const failsWhen = "fails-when"

// jsonNull is the JSON text of null, which a vector states for an output
// that a run does not produce.
const jsonNull = "null"

// checkShrinking runs the one-draw property of a shrinking vector, which
// fails when its predicate holds, and compares the outcome, the valid
// cases, the minimal value, its choices and its token, and the runs that
// shrinking and explaining spent.
func checkShrinking(raw json.RawMessage) error {
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
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	g, err := generatorOf(v.Generator)
	if err != nil {
		return err
	}
	fails, err := predicateOf(v.FailsWhen)
	if err != nil {
		return err
	}
	seed, err := strconv.ParseUint(v.Seed, 10, 64)
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
	if r.Outcome.String() != v.Outcome || r.Cases != v.Cases || r.Runs != v.Runs {
		return fmt.Errorf("the run ends %v after %d cases and %d runs, want %s after %d and %d",
			r.Outcome, r.Cases, r.Runs, v.Outcome, v.Cases, v.Runs)
	}
	if r.Failing == nil {
		if v.Token != nil || string(v.Value) != jsonNull {
			return fmt.Errorf("the run has no minimal case, want %s", v.Value)
		}
		return nil
	}
	if v.Token == nil || r.Token != *v.Token {
		return fmt.Errorf("the token is %s, want %s", r.Token, jsonOf(v.Token))
	}
	minimal := decodedCase{Value: v.Value}
	outcome := engineOutcome{choices: r.Failing.Case.Choices()}
	return minimal.compare(outcome, r.Failing.Case.Draws()[0].Value, v.Choices)
}
