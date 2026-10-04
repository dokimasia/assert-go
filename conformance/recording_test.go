// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The body, the settings and the calls of the recording vectors that the
// tests build.
const (
	// digitBody draws a digit, and fails as big from 5.
	digitBody = `{"draw":` + digitGenerator + `,"fails":[{"identity":"big","when":{"kind":"at-least","n":5}}]}`
	// replaySeven are the settings of a run of the seed 7 that replays the
	// case of the digit 7.
	replaySeven = `{"seed":"7","replay":"prop1:AAc"}`
	// failingSeven are the calls of true of the run of digitBody under
	// replaySeven: one call, which fails.
	failingSeven = `[{"run":1,"phase":"token","verdict":"fail"}]`
	// twoIdentities draws an integer in [0, 20], and fails as odd at an odd
	// value and as big from 10.
	twoIdentities = `{"draw":{"gen":"integer","min":0,"max":20},"fails":[` +
		`{"identity":"odd","when":{"kind":"divisible-by","n":2,"not":true}},` +
		`{"identity":"big","when":{"kind":"at-least","n":10}}]}`
	// twoIdentitiesCalls are the calls of true of the run of twoIdentities
	// under the seed 7, as the definition's executable reference computes
	// them. A body whose failures share one identity makes other calls from
	// the run 10 on.
	twoIdentitiesCalls = `[{"run":1,"phase":"simplest","verdict":"pass"},{"run":2,"phase":"random","verdict":"fail"},` +
		`{"run":3,"phase":"replay","verdict":"fail"},{"run":4,"phase":"shrink","verdict":"pass"},` +
		`{"run":5,"phase":"shrink","verdict":"pass"},{"run":6,"phase":"shrink","verdict":"pass"},` +
		`{"run":7,"phase":"shrink","verdict":"fail"},{"run":8,"phase":"shrink","verdict":"fail"},` +
		`{"run":9,"phase":"shrink","verdict":"fail"},{"run":10,"phase":"shrink","verdict":"fail"},` +
		`{"run":11,"phase":"shrink","verdict":"fail"},{"run":12,"phase":"shrink","verdict":"pass"},` +
		`{"run":13,"phase":"shrink","verdict":"fail"},{"run":14,"phase":"shrink","verdict":"fail"},` +
		`{"run":15,"phase":"explain","verdict":"pass"},{"run":16,"phase":"explain","verdict":"fail"}]`
)

// TestRecording checks a recording vector: the call records of the run of
// a body under its settings, and the example that the settings state. The
// definition's vectors pin the calls of every phase. Written with testing
// rather than with this library, because a verdict is not written with the
// subject.
func TestRecording(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		atExamples := func(segs ...fault.Segment) fault.Path {
			return inVector(append([]fault.Segment{fault.Field(settingsAt), fault.Field("examples")}, segs...)...)
		}
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for the one failing call of a replayed case",
				give: recording(digitBody, replaySeven, "fail", failingSeven),
			},
			{
				name: "returns nil for the calls of a body that fails with two identities",
				give: recording(twoIdentities, seven, "fail", twoIdentitiesCalls),
			},
			{
				name:       "returns a fault at the verdict for a verdict of the property other than the run's",
				give:       recording(digitBody, replaySeven, "pass", failingSeven),
				wantPath:   inVector(fault.Field("verdict")),
				wantReason: `the call record is {"seq":1,"verdict":"fail"}, want {"seq":1,"verdict":"pass"}`,
			},
			{
				name:     "returns a fault at the call for a call of another phase than the run's",
				give:     recording(digitBody, replaySeven, "fail", `[{"run":1,"phase":"random","verdict":"fail"}]`),
				wantPath: inVector(fault.Field("calls"), fault.Index(0)),
				wantReason: `the call record is {"seq":2,"parent":1,"run":1,"phase":"token","verdict":"fail"}, ` +
					`want {"seq":2,"parent":1,"run":1,"phase":"random","verdict":"fail"}`,
			},
			{
				name:       "returns a fault at the calls for fewer calls than the run makes",
				give:       recording(digitBody, replaySeven, "fail", `[]`),
				wantPath:   inVector(fault.Field("calls")),
				wantReason: "the run states 2 call records, want 1",
			},
			{
				name:       "returns a fault at the seed for a seed that is no decimal number",
				give:       recording(digitBody, `{"seed":"seven"}`, "pass", `[]`),
				wantPath:   inVector(fault.Field(settingsAt), fault.Field(seedAt)),
				wantReason: noSeed,
			},
			{
				name:       "returns a fault at the kind for a body of an unknown kind",
				give:       recording(`{"kind":"sleeps"}`, seven, "pass", `[]`),
				wantPath:   inVector(fault.Field("body"), fault.Field(kindAt)),
				wantReason: `"sleeps" names no body`,
			},
			{
				name:       "returns a fault at the examples for two examples",
				give:       recording(digitBody, `{"seed":"7","examples":[[1],[2]]}`, "pass", `[]`),
				wantPath:   atExamples(),
				wantReason: "the settings state 2 examples, and ForAll runs one case of Draws",
			},
			{
				name:       "returns a fault at the examples for an example of a named body",
				give:       recording(`{"kind":"draws-nothing"}`, `{"seed":"7","examples":[[1]]}`, "pass", `[]`),
				wantPath:   atExamples(),
				wantReason: `the body "draws-nothing" states no draw that an example decodes`,
			},
			{
				name:       "returns a fault at the choice of an example for a choice of no stated kind",
				give:       recording(digitBody, `{"seed":"7","examples":[[{}]]}`, "pass", `[]`),
				wantPath:   atExamples(fault.Index(0), fault.Index(0)),
				wantReason: "{} is no choice",
			},
			{
				name: "returns a fault at the example for choices that decode to no value",
				give: recording(`{"draw":{"gen":"filter","of":`+digitGenerator+`,"keep":{"kind":"never"}}}`,
					`{"seed":"7","examples":[[1]]}`, "pass", `[]`),
				wantPath:   atExamples(fault.Index(0)),
				wantReason: "the choices decode to no value",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.CallRecords, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// recording returns a recording vector of the run of body under settings,
// which states the property's verdict and the calls of true.
func recording(body, settings, verdict, calls string) string {
	return fmt.Sprintf(`{"body":%s,"settings":%s,"verdict":%q,"calls":%s}`, body, settings, verdict, calls)
}
