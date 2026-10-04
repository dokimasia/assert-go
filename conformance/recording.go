// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// endContract is the contract of the call of true that ends the body of a
// recording vector.
const endContract = "the body of the recording vector passes"

// recordingSettings are the settings of a recording vector's run: those of
// a behaviour vector, and the choices of each example case.
type recordingSettings struct {
	behaviourSettings

	// Examples are the choices of each example case, each choice in the
	// form of a stored case's.
	Examples [][]json.RawMessage `json:"examples"`
}

// recorded is a call record as a recording vector states it. The vector
// states the run, the phase and the verdict of each call in the property's
// body, and the verdict of the property, whose record has the seq 1. A
// field that a record does not state is the zero value.
type recorded struct {
	Seq     int    `json:"seq"`
	Parent  int    `json:"parent,omitempty"`
	Run     int    `json:"run,omitempty"`
	Phase   string `json:"phase,omitempty"`
	Verdict string `json:"verdict"`
}

// asserting ends a recording vector's case with one call of true, which
// fails when the case fails. A failing case first keeps the record of its
// failure's identity, so the identity that the engine keeps is the one
// that the corpus names, as for a behaviour vector's body.
func asserting(c *prop.Case, failure string) {
	if failure != "" {
		c.Report(assert.Failure{Assertion: failure}, false)
	}
	assert.True(c, failure == "", endContract)
}

// checkRecording runs the body of a recording vector under ForAll on a
// recorder, with its settings, and compares the call records that the
// recorder keeps with the ones the vector states: the property's verdict,
// and the run, the phase and the verdict of each call of true in order. The
// call at position k of the vector, from 0, has the seq k + 2 and the
// parent 1. The vector's example is the case of a Draws entry, whose value
// is the one that the example's choices decode to.
func checkRecording(raw json.RawMessage, dir string) error {
	var v struct {
		// Body is the body spec.
		Body json.RawMessage `json:"body"`
		// Settings are the settings of the run.
		Settings recordingSettings `json:"settings"`
		// Verdict is the property's verdict.
		Verdict string `json:"verdict"`
		// Calls are the calls of true in the property's body, in order.
		Calls []recorded `json:"calls"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	opts, err := v.Settings.resolve(dir)
	if err != nil {
		return fault.At(err, fault.Field(settingsMember))
	}
	body, err := bodyOf(v.Body, asserting)
	if err != nil {
		return fault.At(err, fault.Field(bodyMember))
	}
	example, err := v.Settings.example(v.Body)
	if err != nil {
		return fault.At(err, fault.Field(settingsMember), fault.Field(examplesMember))
	}
	rec := assert.NewRecorder()
	prop.ForAll(rec, behaviourContract, body, append(opts, example...)...)
	want := []recorded{{Seq: 1, Verdict: v.Verdict}}
	for k, c := range v.Calls {
		c.Seq, c.Parent = k+2, 1
		want = append(want, c)
	}
	calls := callsOf(rec)
	got := make([]recorded, len(calls))
	for i, c := range calls {
		got[i] = recorded{Seq: c.Seq, Parent: c.Parent, Run: c.Run, Phase: c.Phase, Verdict: c.Verdict}
	}
	for i := range min(len(got), len(want)) {
		if got[i] == want[i] {
			continue
		}
		path := []fault.Segment{fault.Field(verdictMember)}
		if i > 0 {
			path = []fault.Segment{fault.Field(callsMember), fault.Index(i - 1)}
		}
		return fault.At(fault.New("the call record is %s, want %s", jsonOf(got[i]), jsonOf(want[i])), path...)
	}
	if len(got) != len(want) {
		return fault.At(fault.New("the run states %d call records, want %d", len(got), len(want)),
			fault.Field(callsMember))
	}
	return nil
}

// example returns the option of the case of Draws that states the example
// of s, and no option for a run without one. The entry's value is the one
// that the example's choices decode to under the draw of body, the body
// spec. It returns a fault for more than one example, because ForAll runs
// one case of Draws, for a named body, which states no draw, and at the
// example for choices that are no choices or that decode to no value.
func (s recordingSettings) example(body json.RawMessage) ([]prop.Option, error) {
	if len(s.Examples) == 0 {
		return nil, nil
	}
	if len(s.Examples) > 1 {
		return nil, fault.New("the settings state %d examples, and ForAll runs one case of Draws", len(s.Examples))
	}
	var spec bodySpec
	// bodyOf has parsed the body spec.
	_ = json.Unmarshal(body, &spec)
	if spec.Kind != "" {
		return nil, fault.New("the body %q states no draw that an example decodes", spec.Kind)
	}
	choices, err := parseChoices(s.Examples[0])
	if err != nil {
		return nil, fault.At(err, fault.Index(0))
	}
	// bodyOf has read the generator of the draw.
	g, _ := generatorOf(spec.Draw)
	value, outcome := decodeWith(g, func(body engine.Body) engine.Execution {
		return engine.Replay(body, choices, nil)
	})
	if outcome.rejected {
		return nil, fault.At(fault.New("the choices decode to no value"), fault.Index(0))
	}
	// A generator of the definition decodes a value that a typed literal
	// states.
	lit, _ := literal.Encode(value)
	entries, _ := json.Marshal([]labelled{{Label: drawnLabel, Value: lit}})
	return []prop.Option{prop.Draws(string(entries))}, nil
}
