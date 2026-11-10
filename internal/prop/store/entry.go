// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"encoding/json"
	"fmt"
	"iter"
	"strings"
	"time"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/token"
)

// The formats of the entries that this package writes and replays.
const (
	// Format is the format of an entry whose counterexample states draws
	// alone.
	Format = 1
	// StepsFormat is the format of an entry whose counterexample states the
	// steps of a machine among its draws.
	StepsFormat = 2
)

// The bounds of an entry's form, which keep every language's verdict on a
// file the same.
const (
	// maxLine is the largest line of an identity, the largest signed
	// 32-bit integer.
	maxLine = 2147483647
	// maxClient is the largest client of a step, the largest signed 32-bit
	// integer.
	maxClient = 2147483647
	// maxDepth is the most levels of objects and arrays that a file nests,
	// the root object being the first.
	maxDepth = 64
)

// The spelling of an entry's file name.
const (
	// nameSuffix ends the name of every entry's file.
	nameSuffix = ".json"
	// separators are the characters that a base name does not contain.
	separators = `/\`
)

// Identity is the identity of an entry's failure, in one of four shapes:
//
//   - An assertion's record with a location: Assertion, File and Line.
//   - An assertion's record without a location: Assertion and Contract.
//   - A message that the body passed to the case: File and Line.
//   - An error that the body raised: Error, File and Line.
//
// File is the base name of the failing frame's file, and Line is from 1 to
// 2^31 - 1. Every field outside the shape is empty.
type Identity struct {
	// Assertion is the assertion that reported the failure.
	Assertion string `json:"assertion,omitempty"`
	// Contract is the contract of an assertion's record without a
	// location.
	Contract string `json:"contract,omitempty"`
	// Error is the type of the error that the body raised.
	Error string `json:"error,omitempty"`
	// File is the base name of the failing frame's file.
	File string `json:"file,omitempty"`
	// Line is the failing frame's line.
	Line int `json:"line,omitempty"`
}

// Valid reports whether i has one of the four shapes, with a base name and
// a line from 1 to 2^31 - 1.
func (i Identity) Valid() bool {
	if i.Contract != "" {
		return i.Assertion != "" && i.Error == "" && i.File == "" && i.Line == 0
	}
	if i.Assertion != "" && i.Error != "" {
		return false
	}
	return i.File != "" && !strings.ContainsAny(i.File, separators) && i.Line >= 1 && i.Line <= maxLine
}

// Draw is one value of an entry's counterexample.
type Draw struct {
	// Label is the label that the body drew the value under.
	Label string
	// Value is the typed literal of the value, and nil for a value that no
	// typed literal states.
	Value json.RawMessage
}

// Step is one step of a machine in an entry's counterexample.
type Step struct {
	// Action is the name of the step's action.
	Action string
	// Client is the client of a step of a concurrent section, from 0 to
	// 2^31 - 1, and -1 for any other step.
	Client int
	// Drain reports a step of the drain.
	Drain bool
	// Draws is the number of draws of the counterexample before the step.
	Draws int
}

// Entry is one entry of a store: a property's minimal failing case.
type Entry struct {
	// Definition is the definition version that wrote the entry, as
	// MAJOR.MINOR.PATCH.
	Definition string
	// Property is the property's contract.
	Property string
	// Identity is the identity of the case's failure.
	Identity Identity
	// Choices are the choices of the case.
	Choices []choice.Choice
	// Counterexample are the values that the case drew, in order.
	Counterexample []Draw
	// Steps are the steps of the machine of the case, in order, and empty
	// for a case without one. An entry of [Format] has none.
	Steps []Step
	// Found is when the run wrote the entry. The entry keeps its UTC date.
	Found time.Time
}

// document is an entry as the JSON object of its format, with the fields
// in the order that the definition lists them.
type document struct {
	// Store is the format.
	Store int `json:"store"`
	// Definition is the definition version.
	Definition string `json:"definition"`
	// Property is the property's contract.
	Property string `json:"property"`
	// Identity is the identity of the failure.
	Identity Identity `json:"identity"`
	// Choices is the replay token of the case.
	Choices string `json:"choices"`
	// Counterexample are the draw entries and the step entries in request
	// order, an empty list for none.
	Counterexample []counterexampleEntry `json:"counterexample"`
	// Found is the UTC date, as YYYY-MM-DD.
	Found string `json:"found"`
}

// counterexampleEntry is one entry of a counterexample as its JSON object. A
// draw entry has a label and, when the draw has one, a value. A step entry
// has an action, a client in a concurrent section and true for drain in the
// drain.
type counterexampleEntry struct {
	// Label is the label of a draw entry, and nil for a step entry.
	Label *string `json:"label,omitempty"`
	// Value is the typed literal of a draw entry's value, and nil for a value
	// that no typed literal states and for a step entry.
	Value json.RawMessage `json:"value,omitempty"`
	// Step is the action of a step entry, and nil for a draw entry.
	Step *string `json:"step,omitempty"`
	// Client is the client of a step of a concurrent section, and nil for any
	// other entry.
	Client *int `json:"client,omitempty"`
	// Drain is true for a step of the drain, and false for any other entry.
	Drain bool `json:"drain,omitempty"`
}

// Name returns the name of the entry's file: [random.Mix] of the
// property's contract, a zero byte and the replay token of the choices, as
// 16 lowercase hexadecimal digits followed by ".json".
func (e Entry) Name() string {
	return fmt.Sprintf("%016x%s", random.Mix(e.Property+"\x00"+token.Encode(e.Choices)), nameSuffix)
}

// MarshalJSON returns the entry as the JSON object of [StepsFormat] for an
// entry with steps, and of [Format] for any other entry. The counterexample
// lists each step before the draws that the case drew after it. It returns
// an error when a draw's value is not JSON.
func (e Entry) MarshalJSON() ([]byte, error) {
	format := Format
	if len(e.Steps) > 0 {
		format = StepsFormat
	}
	return json.Marshal(document{
		Store:          format,
		Definition:     e.Definition,
		Property:       e.Property,
		Identity:       e.Identity,
		Choices:        token.Encode(e.Choices),
		Counterexample: e.entries(),
		Found:          e.Found.UTC().Format(time.DateOnly),
	})
}

// Entries returns the draws and the steps of the counterexample in request
// order. Each pair has a draw or a step, and nil in place of the other. The
// steps that the case took before a draw come before it, and the steps
// after the last draw come at the end.
func (e Entry) Entries() iter.Seq2[*Draw, *Step] {
	return func(yield func(*Draw, *Step) bool) {
		steps := e.Steps
		for i := range e.Counterexample {
			for len(steps) > 0 && steps[0].Draws <= i {
				if !yield(nil, &steps[0]) {
					return
				}
				steps = steps[1:]
			}
			if !yield(&e.Counterexample[i], nil) {
				return
			}
		}
		for i := range steps {
			if !yield(nil, &steps[i]) {
				return
			}
		}
	}
}

// entries returns the draws and the steps of the counterexample as the
// entries of its JSON, in request order.
func (e Entry) entries() []counterexampleEntry {
	out := make([]counterexampleEntry, 0, len(e.Counterexample)+len(e.Steps))
	for d, s := range e.Entries() {
		if s == nil {
			out = append(out, counterexampleEntry{Label: &d.Label, Value: d.Value})
			continue
		}
		entry := counterexampleEntry{Step: &s.Action, Drain: s.Drain}
		if s.Client >= 0 {
			entry.Client = &s.Client
		}
		out = append(out, entry)
	}
	return out
}
