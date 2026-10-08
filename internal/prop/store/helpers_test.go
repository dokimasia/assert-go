// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"encoding/json"
	"maps"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/store"
)

// The pins of the entry that the definition's executable reference writes
// for the first store vector of the corpus.
const (
	// contract is the property of the pinned entry.
	contract = "decoding undoes encoding"
	// pinnedName is the name of the pinned entry's file.
	pinnedName = "ccc741d57d7f920d.json"
	// pinnedJSON is the pinned entry as compact JSON, with its fields in the
	// order that the definition lists them.
	pinnedJSON = `{"store":1,"definition":"1.2.0","property":"decoding undoes encoding",` +
		`"identity":{"assertion":"equal","file":"codec_test.go","line":18},` +
		`"choices":"prop1:AAEAAAABAQEAAA",` +
		`"counterexample":[{"label":"values","value":{"type":"list","of":"int","value":[0,-1]}}],` +
		`"found":"2026-10-01"}`
)

// invalidVerdict is the first value past the four verdicts.
const invalidVerdict store.Verdict = 4

// base are the fields of an entry of the property contract whose choices
// are the one integer 7, from the store vectors of the corpus.
var base = map[string]any{
	"store":          1,
	"definition":     "1.2.0",
	"property":       contract,
	"identity":       map[string]any{"assertion": "equal", "file": "codec_test.go", "line": 18},
	"choices":        "prop1:AAc",
	"counterexample": []any{},
	"found":          "2026-10-01",
}

// pinned returns the entry of the first store vector of the corpus, found
// in the morning of 1 October 2026.
func pinned() store.Entry {
	return store.Entry{
		Definition: "1.2.0",
		Property:   contract,
		Identity:   store.Identity{Assertion: "equal", File: "codec_test.go", Line: 18},
		Choices:    integers(1, 0, 1, -1, 0),
		Counterexample: []store.Draw{
			{Label: "values", Value: json.RawMessage(`{"type": "list", "of": "int", "value": [0, -1]}`)},
		},
		Found: time.Date(2026, time.October, 1, 9, 30, 0, 0, time.UTC),
	}
}

// integers returns integer choices of the values.
func integers(values ...int64) []choice.Choice {
	out := make([]choice.Choice, len(values))
	for i, v := range values {
		out[i] = choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(v)}
	}
	return out
}

// entryText returns the JSON of base with changes made and the fields of
// removed taken out, failing the test when it cannot be written.
func entryText(tb testing.TB, changes map[string]any, removed ...string) string {
	tb.Helper()
	fields := maps.Clone(base)
	maps.Copy(fields, changes)
	for _, field := range removed {
		delete(fields, field)
	}
	text, err := json.Marshal(fields)
	assert.NoError(tb, err, "the fields are JSON")
	return string(text)
}

// mustRecordModes skips t on a platform whose file systems record no
// permission bits, such as Windows, which reads a file's mode from its
// read-only attribute and honours no mode of a directory.
func mustRecordModes(t *testing.T) {
	t.Helper()
	if err := filetree.ModesUnrecorded(); err != nil {
		t.Skip(err)
	}
}

// sameChoices reports whether a and b are the same choices in order.
func sameChoices(a, b []choice.Choice) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}
