// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"go.dokimi.dev/assert/internal/prop/store"
)

// dateLayout is the layout of an entry's date of discovery.
const dateLayout = "2006-01-02"

// checkStore reads the text of one file of a store vector and compares the
// verdict and the choices to replay, or writes its entry and compares the
// entry's name and its JSON.
func checkStore(raw json.RawMessage) error {
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
		// Counterexample are the draws to record.
		Counterexample []store.Draw `json:"counterexample"`
		// Definition is the definition version to record.
		Definition string `json:"definition"`
		// Found is the date of discovery to record.
		Found string `json:"found"`
		// Name is the name of the entry's file.
		Name string `json:"name"`
		// Entry is the entry's JSON.
		Entry json.RawMessage `json:"entry"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	if v.Text != nil {
		entry, verdict, _ := store.Read([]byte(*v.Text), v.Contract)
		if verdict.String() != v.Verdict {
			return fmt.Errorf("the verdict is %v, want %s", verdict, v.Verdict)
		}
		if verdict != store.Replay {
			return nil
		}
		same, err := sameChoices(entry.Choices, v.Choices)
		if err != nil {
			return err
		}
		if !same {
			return fmt.Errorf("the entry replays %v, want %s", entry.Choices, jsonOf(v.Choices))
		}
		return nil
	}
	choices, err := parseChoices(v.Choices)
	if err != nil {
		return err
	}
	found, err := time.Parse(dateLayout, v.Found)
	if err != nil {
		return err
	}
	entry := store.Entry{
		Definition:     v.Definition,
		Property:       v.Contract,
		Identity:       v.Identity,
		Choices:        choices,
		Counterexample: v.Counterexample,
		Found:          found,
	}
	if entry.Name() != v.Name {
		return fmt.Errorf("the entry's name is %s, want %s", entry.Name(), v.Name)
	}
	// Each draw's value parsed as JSON with the vector, so the entry
	// marshals.
	data, _ := entry.MarshalJSON()
	if !sameJSON(data, v.Entry) {
		return fmt.Errorf("the entry is %s, want %s", data, v.Entry)
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
