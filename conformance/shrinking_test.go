// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The inputs of the definition's vectors integer/passes-when-no-value-fails
// and integer/shrinks-to-the-lower-boundary-of-the-failing-range, and the
// outputs of the first.
const (
	// wideGenerator is the integer generator over [-1000, 1000].
	wideGenerator = `{"gen":"integer","min":-1000,"max":1000}`
	// fromFiveHundred is the predicate under which a value of 500 or more
	// fails.
	fromFiveHundred = `{"kind":"at-least","n":500}`
	// passedOutputs are the outputs of a run of seed 7 in which no value
	// fails.
	passedOutputs = `"outcome":"passed","cases":100,"value":null,"choices":null,"token":null,"runs":0`
	// fiveHundred is the value of the minimal case of fromFiveHundred.
	fiveHundred = `{"type":"int","value":500}`
	// minimalToken is the JSON text of the token of that minimal case.
	minimalToken = `"prop1:APQD"`
)

// TestShrinking checks a shrinking vector: the run of a property of one
// draw, and the minimal case its failure shrinks to. Written with testing
// rather than with this library, because a verdict is not written with the
// subject.
func TestShrinking(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		atToken := inVector(fault.Field("token"))
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for a vector that states the minimal case",
				give: shrunk(wideGenerator, fromFiveHundred, "7", minimal(fiveHundred, minimalToken)),
			},
			{
				name:       "returns a fault at the generator for a generator of an unknown id",
				give:       shrunk(unknownGenerator, fromFiveHundred, "7", passedOutputs),
				wantPath:   inVector(fault.Field(generatorAt), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at fails-when for a predicate of an unknown kind",
				give:       shrunk(wideGenerator, `{"kind":"most"}`, "7", passedOutputs),
				wantPath:   inVector(fault.Field("fails-when"), fault.Field(kindAt)),
				wantReason: `"most" names no predicate`,
			},
			{
				name: "returns a fault at the outcome for a run of another outcome",
				give: shrunk(wideGenerator, `{"kind":"never"}`, "7",
					`"outcome":"counterexample","cases":100,"value":null,"choices":null,"token":null,"runs":0`),
				wantPath:   inVector(fault.Field("outcome")),
				wantReason: "the run ends as passed, want counterexample",
			},
			{
				name: "returns a fault at the cases for a run of another number of valid cases",
				give: shrunk(wideGenerator, `{"kind":"never"}`, "7",
					`"outcome":"passed","cases":99,"value":null,"choices":null,"token":null,"runs":0`),
				wantPath:   inVector(fault.Field(casesAt)),
				wantReason: "the run has 100 valid cases, want 99",
			},
			{
				name: "returns a fault at the runs for a run of another number of shrink runs",
				give: shrunk(wideGenerator, fromFiveHundred, "7",
					`"outcome":"counterexample","cases":4,"value":`+fiveHundred+`,"choices":[500],`+
						`"token":`+minimalToken+`,"runs":14`),
				wantPath:   inVector(fault.Field("runs")),
				wantReason: "shrinking and explaining spend 15 runs, want 14",
			},
			{
				name: "returns a fault at the value for a passing run that states a minimal value",
				give: shrunk(wideGenerator, `{"kind":"never"}`, "7",
					`"outcome":"passed","cases":100,"value":`+fiveHundred+`,"choices":null,"token":null,"runs":0`),
				wantPath:   inVector(fault.Field(valueAt)),
				wantReason: "the run has no minimal case, want " + fiveHundred,
			},
			{
				name: "returns a fault at the token for a passing run that states a token",
				give: shrunk(wideGenerator, `{"kind":"never"}`, "7",
					`"outcome":"passed","cases":100,"value":null,"choices":null,"token":"prop1:","runs":0`),
				wantPath:   atToken,
				wantReason: "the run has no minimal case, want the token prop1:",
			},
			{
				name:       "returns a fault at the token for a minimal case of another token",
				give:       shrunk(wideGenerator, fromFiveHundred, "7", minimal(fiveHundred, `"prop1:AAA"`)),
				wantPath:   atToken,
				wantReason: `the token is prop1:APQD, want "prop1:AAA"`,
			},
			{
				name:       "returns a fault at the token for a minimal case without a token",
				give:       shrunk(wideGenerator, fromFiveHundred, "7", minimal(fiveHundred, null)),
				wantPath:   atToken,
				wantReason: "the token is prop1:APQD, want null",
			},
			{
				name: "returns a fault at the value for a minimal case of another value",
				give: shrunk(wideGenerator, fromFiveHundred, "7",
					minimal(`{"type":"int","value":501}`, minimalToken)),
				wantPath:   inVector(fault.Field(valueAt)),
				wantReason: `the value is int:500, want {"type":"int","value":501}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Shrinking, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// shrunk returns a shrinking vector of generator that fails under the
// predicate failsWhen in a run of seed, and states outputs, the JSON
// members of its outputs.
func shrunk(generator, failsWhen, seed, outputs string) string {
	return fmt.Sprintf(`{"generator":%s,"fails-when":%s,"seed":%q,%s}`, generator, failsWhen, seed, outputs)
}

// minimal returns the outputs of the run of seed 7 that fails from 500,
// with the typed literal value and the JSON text token of its minimal
// case.
func minimal(value, token string) string {
	return fmt.Sprintf(`"outcome":"counterexample","cases":4,"value":%s,"choices":[500],"token":%s,"runs":15`,
		value, token)
}
