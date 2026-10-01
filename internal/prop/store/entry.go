// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/token"
)

// Format is the format of the entries that this package writes and
// replays.
const Format = 1

// The bounds of an entry's form, which keep every language's verdict on a
// file the same.
const (
	// maxLine is the largest line of an identity, the largest signed
	// 32-bit integer.
	maxLine = 2147483647
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
	Label string `json:"label"`
	// Value is the typed literal of the value, and nil for a value that no
	// typed literal states.
	Value json.RawMessage `json:"value,omitempty"`
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
	// Counterexample are the drawn values, an empty list for none.
	Counterexample []Draw `json:"counterexample"`
	// Found is the UTC date, as YYYY-MM-DD.
	Found string `json:"found"`
}

// Name returns the name of the entry's file: [random.Mix] of the
// property's contract, a zero byte and the replay token of the choices, as
// 16 lowercase hexadecimal digits followed by ".json".
func (e Entry) Name() string {
	return fmt.Sprintf("%016x%s", random.Mix(e.Property+"\x00"+token.Encode(e.Choices)), nameSuffix)
}

// MarshalJSON returns the entry as the JSON object of [Format]. It returns
// an error when a draw's value is not JSON.
func (e Entry) MarshalJSON() ([]byte, error) {
	counterexample := e.Counterexample
	if counterexample == nil {
		counterexample = []Draw{}
	}
	return json.Marshal(document{
		Store:          Format,
		Definition:     e.Definition,
		Property:       e.Property,
		Identity:       e.Identity,
		Choices:        token.Encode(e.Choices),
		Counterexample: counterexample,
		Found:          e.Found.UTC().Format(time.DateOnly),
	})
}
