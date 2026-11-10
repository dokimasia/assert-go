// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"reflect"
	"time"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/store"
)

// dateLayout is the layout of an entry's date of discovery.
const dateLayout = "2006-01-02"

// checkStore reads the text of one file of a store vector and compares the
// verdict and the choices to replay, or writes its entry and compares the
// entry's name and its JSON.
func checkStore(raw json.RawMessage, _ string) error {
	var v struct {
		// Contract is the property's contract.
		Contract string `json:"contract"`
		// Text is the text of a file to read, and nil for a write.
		Text *string `json:"text"`
		// Verdict is the verdict on the text.
		Verdict string `json:"verdict"`
		// Choices are the choices to record, or the choices to replay.
		Choices []json.RawMessage `json:"choices"`
		// Identity is the identity of the failure to record.
		Identity store.Identity `json:"identity"`
		// Counterexample are the draws and the steps to record.
		Counterexample recordedCounterexample `json:"counterexample"`
		// Definition is the definition version to record.
		Definition string `json:"definition"`
		// Found is the date of discovery to record.
		Found string `json:"found"`
		// Name is the name of the entry's file.
		Name string `json:"name"`
		// Entry is the entry's JSON.
		Entry json.RawMessage `json:"entry"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	if v.Text != nil {
		entry, verdict, _ := store.Read([]byte(*v.Text), v.Contract)
		if verdict.String() != v.Verdict {
			return fault.At(fault.New("the verdict is %v, want %s", verdict, v.Verdict), fault.Field(verdictMember))
		}
		if verdict != store.Replay {
			return nil
		}
		return at(compareChoices(entry.Choices, v.Choices), fault.Field(choicesMember))
	}
	choices, err := parseChoices(v.Choices)
	if err != nil {
		return fault.At(err, fault.Field(choicesMember))
	}
	found, err := time.Parse(dateLayout, v.Found)
	if err != nil {
		return fault.At(fault.New("%q is no date", v.Found).Because(err), fault.Field(foundMember))
	}
	entry := store.Entry{
		Definition:     v.Definition,
		Property:       v.Contract,
		Identity:       v.Identity,
		Choices:        choices,
		Counterexample: v.Counterexample.draws,
		Steps:          v.Counterexample.steps,
		Found:          found,
	}
	if entry.Name() != v.Name {
		return fault.At(fault.New("the entry's name is %s, want %s", entry.Name(), v.Name), fault.Field(nameMember))
	}
	// Each draw's value parsed as JSON with the vector, so the entry
	// marshals.
	data, _ := entry.MarshalJSON()
	if !sameJSON(data, v.Entry) {
		return fault.At(fault.New("the entry is %s, want %s", data, v.Entry), fault.Field(entryMember))
	}
	return nil
}

// recordedCounterexample is the counterexample that an entry of a store
// records, as a vector states it: draw entries, each with a label and the
// value where the entry records one, and step entries, each with an action,
// a client in a concurrent section and true for drain in the drain.
type recordedCounterexample struct {
	// draws are the draws of the counterexample, in order.
	draws []store.Draw
	// steps are the steps of the counterexample, each with the number of
	// draws before it.
	steps []store.Step
}

// UnmarshalJSON reads the entries of a counterexample in request order. It
// returns a fault for a counterexample that is no list of objects, and one
// at the index of an entry that does not state a label or a step.
func (r *recordedCounterexample) UnmarshalJSON(data []byte) error {
	var entries []struct {
		Label  *string         `json:"label"`
		Value  json.RawMessage `json:"value"`
		Step   *string         `json:"step"`
		Client *int            `json:"client"`
		Drain  bool            `json:"drain"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return fault.New("the counterexample is no list of entries").Because(err)
	}
	for i, e := range entries {
		switch {
		case e.Step != nil:
			step := store.Step{Action: *e.Step, Client: -1, Drain: e.Drain, Draws: len(r.draws)}
			if e.Client != nil {
				step.Client = *e.Client
			}
			r.steps = append(r.steps, step)
		case e.Label != nil:
			r.draws = append(r.draws, store.Draw{Label: *e.Label, Value: e.Value})
		default:
			return fault.At(fault.New("the entry does not state a label or a step"), fault.Index(i))
		}
	}
	return nil
}

// sameJSON reports whether a and b, two JSON texts, state one value.
func sameJSON(a, b []byte) bool {
	var x, y any
	_ = json.Unmarshal(a, &x)
	_ = json.Unmarshal(b, &y)
	return reflect.DeepEqual(x, y)
}
