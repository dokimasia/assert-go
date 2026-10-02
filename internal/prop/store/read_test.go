// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/store"
)

// invalidVerdict is the first value past the four verdicts.
const invalidVerdict store.Verdict = 4

// readAllocs are the allocations of Read on the pinned entry, measured.
const readAllocs = 135

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

// TestRead checks the verdict on every kind of file, pinned to the store
// vectors of the corpus, and each verdict's spelling.
func TestRead(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give store.Verdict
			want bool
		}{
			{name: "reports true for Replay", give: store.Replay, want: true},
			{name: "reports true for Damaged", give: store.Damaged, want: true},
			{name: "reports false past Damaged", give: invalidVerdict, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a verdict")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give store.Verdict
			want string
		}{
			{name: "returns replay for Replay", give: store.Replay, want: "replay"},
			{name: "returns other for Other", give: store.Other, want: "other"},
			{name: "returns skip for Skip", give: store.Skip, want: "skip"},
			{name: "returns damaged for Damaged", give: store.Damaged, want: "damaged"},
			{name: "returns Verdict(4) for a value that is no verdict", give: invalidVerdict, want: "Verdict(4)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the verdict's spelling")
			})
		}
	})

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		replayed := []struct {
			name string
			give string
			want store.Verdict
		}{
			{name: "returns Replay for an entry of the property", give: entryText(t, nil), want: store.Replay},
			{
				name: "returns Replay for an entry of another definition version",
				give: entryText(t, map[string]any{"definition": "1.1.0"}),
				want: store.Replay,
			},
			{
				name: "returns Other for an entry of another property",
				give: entryText(t, map[string]any{"property": "encoding is deterministic"}),
				want: store.Other,
			},
			{
				name: "returns Replay for a draw without a value",
				give: withDraws(t, map[string]any{"label": "conn"}),
				want: store.Replay,
			},
			{
				name: "returns Replay for a draw whose value is no typed literal",
				give: withDraws(t, map[string]any{"label": "x", "value": []any{1}}),
				want: store.Replay,
			},
			{
				name: "returns Replay for a label that is the replacement character",
				give: withDraws(t, map[string]any{"label": "�"}),
				want: store.Replay,
			},
			{
				name: "returns Replay for an assertion's record without a location",
				give: withIdentity(t, map[string]any{"assertion": "equal", "contract": "x"}),
				want: store.Replay,
			},
			{
				name: "returns Replay for a message at the largest line",
				give: withIdentity(t, map[string]any{"file": "a_test.go", "line": 2147483647}),
				want: store.Replay,
			},
			{
				name: "returns Replay for a raised error",
				give: withIdentity(t, map[string]any{"error": "string", "file": "a.go", "line": 1}),
				want: store.Replay,
			},
			{
				name: "returns Replay for an entry nested 64 levels",
				give: withDraws(t, map[string]any{"label": "v", "value": nested(61)}),
				want: store.Replay,
			},
			{
				name: "returns Replay for an entry found in year 1",
				give: entryText(t, map[string]any{"found": "0001-01-01"}),
				want: store.Replay,
			},
		}
		for _, tt := range replayed {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e, verdict, err := store.Read([]byte(tt.give), contract)
				assert.NoError(t, err, "the file is an entry")
				assert.Equal(t, verdict, tt.want, "the verdict")
				assert.True(t, sameChoices(e.Choices, integers(7)), "the choices of prop1:AAc")
			})
		}

		t.Run("returns the fields that the entry states", func(t *testing.T) {
			t.Parallel()
			text, err := json.Marshal(pinned())
			assert.NoError(t, err, "the pinned entry is JSON")
			e, verdict, err := store.Read(text, contract)
			assert.NoError(t, err, "the pinned entry")
			assert.Equal(t, verdict, store.Replay, "an entry of the property")
			assert.Equal(t, e.Definition, "1.2.0", "the definition version")
			wantIdentity := store.Identity{Assertion: "equal", File: "codec_test.go", Line: 18}
			assert.Equal(t, e.Identity, wantIdentity, "the identity")
			assert.True(t, sameChoices(e.Choices, integers(1, 0, 1, -1, 0)), "the choices")
			assert.Equal(t, e.Counterexample[0].Label, "values", "the draw's label")
			wantValue := `{"type":"list","of":"int","value":[0,-1]}`
			assert.Equal(t, string(e.Counterexample[0].Value), wantValue, "the draw's value")
			assert.Equal(t, e.Found.Format(time.DateOnly), "2026-10-01", "the date")
		})

		t.Run("returns the saturated choices of a token whose element is 2^32 or more", func(t *testing.T) {
			t.Parallel()
			text := entryText(t, map[string]any{"choices": "prop1:AwGAgICAEA"})
			e, verdict, err := store.Read([]byte(text), contract)
			assert.NoError(t, err, "the definition accepts every element below 2^64")
			assert.Equal(t, verdict, store.Replay, "an entry of the property")
			want := []choice.Choice{{Kind: choice.Sequence, Sequence: []uint32{4294967295}}}
			assert.True(t, sameChoices(e.Choices, want), "the element saturates")
		})

		skipped := []struct {
			name  string
			give  string
			fault string
		}{
			{
				name:  "returns Skip for an entry of a later format",
				give:  `{"store": 2, "layout": "unknown"}`,
				fault: "format 2",
			},
			{
				name:  "returns Skip for a format beyond 64 bits",
				give:  `{"store": 100000000000000000000}`,
				fault: "format 100000000000000000000",
			},
			{
				name:  "returns Skip for a token of a later version",
				give:  entryText(t, map[string]any{"choices": "prop2:AAc"}),
				fault: "the token is of version 2",
			},
		}
		for _, tt := range skipped {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e, verdict, err := store.Read([]byte(tt.give), contract)
				assert.Equal(t, verdict, store.Skip, "the verdict")
				assert.ErrorIs(t, err, store.ErrLater, "the reason to skip")
				assert.Contains(t, err.Error(), tt.fault, "the reason")
				assert.Equal(t, e.Property, "", "no entry")
			})
		}

		repeated := strings.Replace(entryText(t, nil), `"choices":"prop1:AAc"`,
			`"choices":"prop1:AAc","choices":"prop1:AAM"`, 1)
		huge := json.Number("1" + strings.Repeat("0", 30))
		damaged := []struct {
			name  string
			give  string
			fault string
		}{
			{name: "returns Damaged for text that is not JSON", give: "store: 1", fault: "invalid character"},
			{name: "returns Damaged for an empty file", give: "", fault: "states 0 JSON values"},
			{name: "returns Damaged for two values", give: "{} {}", fault: "states 2 JSON values"},
			{name: "returns Damaged for text that is not UTF-8", give: "\xff", fault: "not UTF-8 from byte 0"},
			{
				name:  "returns Damaged naming the first byte that is not UTF-8",
				give:  "{\"store\": \"é\xff\"}",
				fault: "not UTF-8 from byte 13",
			},
			{name: "returns Damaged for a JSON array", give: "[1]", fault: "not one JSON object"},
			{name: "returns Damaged for JSON null", give: "null", fault: "not one JSON object"},
			{name: "returns Damaged for NaN", give: `{"store": NaN}`, fault: "invalid character"},
			{name: "returns Damaged for a repeated name", give: repeated, fault: `repeats the name "choices"`},
			{
				name:  "returns Damaged for a name repeated inside a nested object",
				give:  `{"store": 1, "identity": {"line": 1, "line": 2}}`,
				fault: `repeats the name "line"`,
			},
			{
				name:  "returns Damaged for an entry nested 65 levels",
				give:  withDraws(t, map[string]any{"label": "v", "value": nested(62)}),
				fault: "nests past 64 levels",
			},
			{
				name:  "returns Damaged for nesting far past the bound",
				give:  strings.Repeat("[", 100000) + strings.Repeat("]", 100000),
				fault: "nests past 64 levels",
			},
			{name: "returns Damaged for no store", give: `{"choices": "prop1:AAc"}`, fault: "no format as store"},
			{name: "returns Damaged for store 0", give: `{"store": 0}`, fault: "no format as store"},
			{
				name:  "returns Damaged for a store that is a bool",
				give:  `{"store": true}`,
				fault: "no format as store",
			},
			{
				name:  "returns Damaged for a store with a fraction",
				give:  `{"store": 1.0}`,
				fault: "no format as store",
			},
			{
				name:  "returns Damaged for a store that is a string",
				give:  `{"store": "1"}`,
				fault: "no format as store",
			},
			{name: "returns Damaged for a missing field", give: entryText(t, nil, "found"), fault: "lacks found"},
			{
				name:  "returns Damaged for a field the format does not have",
				give:  entryText(t, map[string]any{"comment": "found in review"}),
				fault: "states 8 fields",
			},
			{
				name:  "returns Damaged for a definition of two numbers",
				give:  entryText(t, map[string]any{"definition": "1.2"}),
				fault: "definition",
			},
			{
				name:  "returns Damaged for a definition with a leading zero",
				give:  entryText(t, map[string]any{"definition": "1.02.0"}),
				fault: "definition",
			},
			{
				name:  "returns Damaged for a definition that is no string",
				give:  entryText(t, map[string]any{"definition": 1}),
				fault: "definition",
			},
			{
				name:  "returns Damaged for a property that is no string",
				give:  entryText(t, map[string]any{"property": 5}),
				fault: "property 5 is no string",
			},
			{
				name:  "returns Damaged for an identity of no shape",
				give:  withIdentity(t, map[string]any{"assertion": "equal"}),
				fault: "has no shape",
			},
			{
				name:  "returns Damaged for an identity with a key of no shape",
				give:  withIdentity(t, map[string]any{"file": "a.go", "line": 1, "column": 2}),
				fault: `states the key "column"`,
			},
			{
				name:  "returns Damaged for line 0",
				give:  withIdentity(t, map[string]any{"file": "a.go", "line": 0}),
				fault: "line 0 is no line",
			},
			{
				name:  "returns Damaged for a line past 2^31 - 1",
				give:  withIdentity(t, map[string]any{"file": "a.go", "line": 2147483648}),
				fault: "line 2147483648 is no line",
			},
			{
				name:  "returns Damaged for a line beyond 64 bits",
				give:  withIdentity(t, map[string]any{"file": "a.go", "line": huge}),
				fault: "is no line",
			},
			{
				name:  "returns Damaged for a line with a fraction",
				give:  withIdentity(t, map[string]any{"file": "a.go", "line": json.RawMessage("3.0")}),
				fault: "line 3.0 is no line",
			},
			{
				name:  "returns Damaged for a line that is a bool",
				give:  withIdentity(t, map[string]any{"file": "a.go", "line": true}),
				fault: "line true is no line",
			},
			{
				name:  "returns Damaged for a file with a slash",
				give:  withIdentity(t, map[string]any{"file": "pkg/a.go", "line": 1}),
				fault: "has no shape",
			},
			{
				name:  "returns Damaged for an empty file name",
				give:  withIdentity(t, map[string]any{"file": "", "line": 1}),
				fault: "no non-empty string",
			},
			{
				name:  "returns Damaged for a file that is no string",
				give:  withIdentity(t, map[string]any{"file": 7, "line": 1}),
				fault: "no non-empty string",
			},
			{
				name:  "returns Damaged for an identity that is no object",
				give:  entryText(t, map[string]any{"identity": "equal"}),
				fault: "is no object",
			},
			{
				name:  "returns Damaged for a null identity",
				give:  entryText(t, map[string]any{"identity": nil}),
				fault: "identity null is no object",
			},
			{
				name:  "returns Damaged for choices that are no string",
				give:  entryText(t, map[string]any{"choices": 5}),
				fault: "choices 5 is no string",
			},
			{
				name:  "returns Damaged for choices without a token version",
				give:  entryText(t, map[string]any{"choices": "AAc"}),
				fault: "states no token version",
			},
			{
				name:  "returns Damaged for a token version with a leading zero",
				give:  entryText(t, map[string]any{"choices": "prop01:AAc"}),
				fault: "states no token version",
			},
			{
				name:  "returns Damaged for a token not in canonical form",
				give:  entryText(t, map[string]any{"choices": "prop1:AAc="}),
				fault: "not unpadded base64url",
			},
			{
				name:  "returns Damaged for a counterexample that is no array",
				give:  entryText(t, map[string]any{"counterexample": map[string]any{"label": "x"}}),
				fault: "is no array",
			},
			{
				name:  "returns Damaged for a null counterexample",
				give:  entryText(t, map[string]any{"counterexample": nil}),
				fault: "counterexample null is no array",
			},
			{
				name:  "returns Damaged for a draw that is no object",
				give:  withDraws(t, "x"),
				fault: `draw 0 "x" is no object`,
			},
			{
				name:  "returns Damaged for a draw without a label",
				give:  withDraws(t, map[string]any{"value": 1}),
				fault: "has no string label",
			},
			{
				name:  "returns Damaged for a label that is no string",
				give:  withDraws(t, map[string]any{"label": 5}),
				fault: "has no string label",
			},
			{
				name:  "returns Damaged for a draw with another key",
				give:  withDraws(t, map[string]any{"label": "x", "note": "y"}),
				fault: "states a key other than label and value",
			},
			{
				name:  "returns Damaged for a day the calendar does not have",
				give:  entryText(t, map[string]any{"found": "2026-02-30"}),
				fault: `found "2026-02-30" is no date`,
			},
			{
				name:  "returns Damaged for a date in words",
				give:  entryText(t, map[string]any{"found": "1 October 2026"}),
				fault: `found "1 October 2026" is no date`,
			},
			{
				name:  "returns Damaged for a date as a number",
				give:  entryText(t, map[string]any{"found": 20261001}),
				fault: "found 20261001 is no date",
			},
			{
				name:  "returns Damaged for year 0",
				give:  entryText(t, map[string]any{"found": "0000-01-01"}),
				fault: `found "0000-01-01" is no date`,
			},
		}
		for _, tt := range damaged {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e, verdict, err := store.Read([]byte(tt.give), contract)
				assert.Equal(t, verdict, store.Damaged, "the verdict")
				assert.ErrorIs(t, err, store.ErrDamaged, "the damage")
				assert.Contains(t, err.Error(), tt.fault, "the fault")
				assert.Equal(t, e.Property, "", "no entry")
			})
		}
	})
}

// TestReadZeroAlloc checks that Valid and String allocate nothing, and the
// ceiling of Read.
func TestReadZeroAlloc(t *testing.T) {
	text, err := json.Marshal(pinned())
	assert.NoError(t, err, "the pinned entry is JSON")
	assert.MaxAllocs(t, func() { _ = store.Damaged.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = store.Damaged.String() }, 0, "String allocates nothing")
	assert.MaxAllocs(t, func() { _, _, _ = store.Read(text, contract) }, readAllocs, "Read allocates its JSON")
}

// BenchmarkRead measures each method of Verdict, and Read on the pinned
// entry.
func BenchmarkRead(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = store.Damaged.Valid()
		}
		assert.True(b, got, "Damaged is a verdict")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = store.Damaged.String()
		}
		assert.Equal(b, got, "damaged", "the verdict's spelling")
	})

	b.Run("Read", func(b *testing.B) {
		text, err := json.Marshal(pinned())
		assert.NoError(b, err, "the pinned entry is JSON")
		var got store.Verdict
		c := bench.Start(b).MaxAllocs(readAllocs)
		defer c.End()
		for c.Loop() {
			_, got, _ = store.Read(text, contract)
		}
		assert.Equal(b, got, store.Replay, "an entry of the property")
	})
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

// withIdentity returns the JSON of base with identity as its identity.
func withIdentity(tb testing.TB, identity map[string]any) string {
	tb.Helper()
	return entryText(tb, map[string]any{"identity": identity})
}

// withDraws returns the JSON of base with draws as its counterexample.
func withDraws(tb testing.TB, draws ...any) string {
	tb.Helper()
	return entryText(tb, map[string]any{"counterexample": draws})
}

// nested returns a list that nests levels arrays deep, the outer one
// included.
func nested(levels int) any {
	var node any = []any{}
	for range levels - 1 {
		node = []any{node}
	}
	return node
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
