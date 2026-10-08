// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/prop/token"
)

// readAllocs are the allocations of Read on the pinned entry, measured.
const readAllocs = 135

// FuzzRead checks that Read returns one of its four verdicts for any
// bytes, and does not panic. The error wraps ErrLater for Skip and
// ErrDamaged for Damaged and is nil otherwise, and only Replay returns
// an entry of the contract.
func FuzzRead(f *testing.F) {
	pinned, err := json.Marshal(base)
	if err != nil {
		f.Fatalf("the pinned entry encodes: %v", err)
	}
	f.Add(pinned)
	f.Add([]byte(`{"store": 2}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`{"store": 1, "store": 1}`))
	f.Add([]byte("{\"store\": \"\xff\"}"))
	f.Fuzz(func(t *testing.T, data []byte) {
		e, verdict, err := store.Read(data, contract)
		switch verdict {
		case store.Replay, store.Other:
			if err != nil || (e.Property == contract) != (verdict == store.Replay) {
				t.Fatalf("Read returns %v for the property %q, with %v", verdict, e.Property, err)
			}
		case store.Skip:
			if !errors.Is(err, store.ErrLater) {
				t.Fatalf("Read skips with %v, which wraps no ErrLater", err)
			}
		case store.Damaged:
			if !errors.Is(err, store.ErrDamaged) {
				t.Fatalf("Read reports damage with %v, which wraps no ErrDamaged", err)
			}
		default:
			t.Fatalf("Read returns the verdict %d, which is none of the four", verdict)
		}
	})
}

// TestRead checks the verdict on every kind of file, pinned to the store
// vectors of the corpus, and the fault of each file that a run skips or
// that is damaged.
func TestRead(t *testing.T) {
	t.Parallel()

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

		atStore := fault.Path{fault.Field("store")}
		atChoices := fault.Path{fault.Field("choices")}
		skipped := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns Skip for an entry of a later format",
				give:       `{"store": 2, "layout": "unknown"}`,
				wantPath:   atStore,
				wantReason: "the entry is of format 2",
			},
			{
				name:       "returns Skip for a format beyond 64 bits",
				give:       `{"store": 100000000000000000000}`,
				wantPath:   atStore,
				wantReason: "the entry is of format 100000000000000000000",
			},
			{
				name:       "returns Skip for a token of a later version",
				give:       entryText(t, map[string]any{"choices": "prop2:AAc"}),
				wantPath:   atChoices,
				wantReason: "the token is of version 2",
			},
		}
		for _, tt := range skipped {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e, verdict, err := store.Read([]byte(tt.give), contract)
				assert.Equal(t, verdict, store.Skip, "the verdict")
				assert.ErrorIs(t, err, store.ErrLater, "the reason to skip")
				f := assert.ErrorAs[*fault.Error](t, err, "a fault")
				assert.Equal(t, f.Path, tt.wantPath, "the field that is later")
				assert.Equal(t, f.Reason, tt.wantReason, "what is later")
				assert.Equal(t, e.Property, "", "no entry")
			})
		}

		repeated := strings.Replace(entryText(t, nil), `"choices":"prop1:AAc"`,
			`"choices":"prop1:AAc","choices":"prop1:AAM"`, 1)
		huge := json.Number("1" + strings.Repeat("0", 30))
		atIdentity := fault.Path{fault.Field("identity")}
		atLine := fault.Path{fault.Field("identity"), fault.Field("line")}
		atFile := fault.Path{fault.Field("identity"), fault.Field("file")}
		atCounterexample := fault.Path{fault.Field("counterexample")}
		atFirstDraw := fault.Path{fault.Field("counterexample"), fault.Index(0)}
		atFound := fault.Path{fault.Field("found")}
		const (
			noShape  = "the keys form none of the four shapes of an identity"
			noFormat = "the entry states no format"
		)
		damaged := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{name: "returns Damaged for text that is not JSON", give: "store: 1", wantReason: "the file is no JSON"},
			{
				name:       "returns Damaged for an empty file",
				give:       "",
				wantReason: "the file states 0 JSON values, not one",
			},
			{
				name:       "returns Damaged for two values",
				give:       "{} {}",
				wantReason: "the file states 2 JSON values, not one",
			},
			{
				name:       "returns Damaged for text that is not UTF-8",
				give:       "\xff",
				wantReason: "the file is not UTF-8 from byte 0",
			},
			{
				name:       "returns Damaged naming the first byte that is not UTF-8",
				give:       "{\"store\": \"é\xff\"}",
				wantReason: "the file is not UTF-8 from byte 13",
			},
			{name: "returns Damaged for a JSON array", give: "[1]", wantReason: "the file is not one JSON object"},
			{name: "returns Damaged for JSON null", give: "null", wantReason: "the file is not one JSON object"},
			{name: "returns Damaged for NaN", give: `{"store": NaN}`, wantReason: "the file is no JSON"},
			{
				name:       "returns Damaged for a repeated name",
				give:       repeated,
				wantReason: `an object repeats the name "choices"`,
			},
			{
				name:       "returns Damaged for a name repeated inside a nested object",
				give:       `{"store": 1, "identity": {"line": 1, "line": 2}}`,
				wantReason: `an object repeats the name "line"`,
			},
			{
				name:       "returns Damaged for an entry nested 65 levels",
				give:       withDraws(t, map[string]any{"label": "v", "value": nested(62)}),
				wantReason: "the file nests past 64 levels",
			},
			{
				name:       "returns Damaged for nesting far past the bound",
				give:       strings.Repeat("[", 100000) + strings.Repeat("]", 100000),
				wantReason: "the file nests past 64 levels",
			},
			{
				name:       "returns Damaged for no store",
				give:       `{"choices": "prop1:AAc"}`,
				wantPath:   atStore,
				wantReason: noFormat,
			},
			{name: "returns Damaged for store 0", give: `{"store": 0}`, wantPath: atStore, wantReason: noFormat},
			{
				name:       "returns Damaged for a store that is a bool",
				give:       `{"store": true}`,
				wantPath:   atStore,
				wantReason: noFormat,
			},
			{
				name:       "returns Damaged for a store with a fraction",
				give:       `{"store": 1.0}`,
				wantPath:   atStore,
				wantReason: noFormat,
			},
			{
				name:       "returns Damaged for a store that is a string",
				give:       `{"store": "1"}`,
				wantPath:   atStore,
				wantReason: noFormat,
			},
			{
				name:       "returns Damaged for a missing field",
				give:       entryText(t, nil, "found"),
				wantPath:   atFound,
				wantReason: "the entry lacks the field",
			},
			{
				name:       "returns Damaged for a field the format does not have",
				give:       entryText(t, map[string]any{"comment": "found in review"}),
				wantReason: "the entry states 8 fields, not the 7 of format 1",
			},
			{
				name:       "returns Damaged for a definition of two numbers",
				give:       entryText(t, map[string]any{"definition": "1.2"}),
				wantPath:   fault.Path{fault.Field("definition")},
				wantReason: `"1.2" is no version`,
			},
			{
				name:       "returns Damaged for a definition with a leading zero",
				give:       entryText(t, map[string]any{"definition": "1.02.0"}),
				wantPath:   fault.Path{fault.Field("definition")},
				wantReason: `"1.02.0" is no version`,
			},
			{
				name:       "returns Damaged for a definition that is no string",
				give:       entryText(t, map[string]any{"definition": 1}),
				wantPath:   fault.Path{fault.Field("definition")},
				wantReason: "1 is no version",
			},
			{
				name:       "returns Damaged for a property that is no string",
				give:       entryText(t, map[string]any{"property": 5}),
				wantPath:   fault.Path{fault.Field("property")},
				wantReason: "5 is no string",
			},
			{
				name:       "returns Damaged for an identity of no shape",
				give:       withIdentity(t, map[string]any{"assertion": "equal"}),
				wantPath:   atIdentity,
				wantReason: noShape,
			},
			{
				name:       "returns Damaged for an identity with a key of no shape",
				give:       withIdentity(t, map[string]any{"file": "a.go", "line": 1, "column": 2}),
				wantPath:   fault.Path{fault.Field("identity"), fault.Field("column")},
				wantReason: "the key is no key of an identity",
			},
			{
				name:       "returns Damaged for line 0",
				give:       withIdentity(t, map[string]any{"file": "a.go", "line": 0}),
				wantPath:   atLine,
				wantReason: "0 is no line",
			},
			{
				name:       "returns Damaged for a line past 2^31 - 1",
				give:       withIdentity(t, map[string]any{"file": "a.go", "line": 2147483648}),
				wantPath:   atLine,
				wantReason: "2147483648 is no line",
			},
			{
				name:       "returns Damaged for a line beyond 64 bits",
				give:       withIdentity(t, map[string]any{"file": "a.go", "line": huge}),
				wantPath:   atLine,
				wantReason: string(huge) + " is no line",
			},
			{
				name:       "returns Damaged for a line with a fraction",
				give:       withIdentity(t, map[string]any{"file": "a.go", "line": json.RawMessage("3.0")}),
				wantPath:   atLine,
				wantReason: "3.0 is no line",
			},
			{
				name:       "returns Damaged for a line that is a bool",
				give:       withIdentity(t, map[string]any{"file": "a.go", "line": true}),
				wantPath:   atLine,
				wantReason: "true is no line",
			},
			{
				name:       "returns Damaged for a line that is an object, without its text",
				give:       withIdentity(t, map[string]any{"file": "a.go", "line": map[string]any{"value": 3}}),
				wantPath:   atLine,
				wantReason: "the value is no line",
			},
			{
				name:       "returns Damaged for a file with a slash",
				give:       withIdentity(t, map[string]any{"file": "pkg/a.go", "line": 1}),
				wantPath:   atIdentity,
				wantReason: noShape,
			},
			{
				name:       "returns Damaged for an empty file name",
				give:       withIdentity(t, map[string]any{"file": "", "line": 1}),
				wantPath:   atFile,
				wantReason: `"" is no non-empty string`,
			},
			{
				name:       "returns Damaged for a file that is no string",
				give:       withIdentity(t, map[string]any{"file": 7, "line": 1}),
				wantPath:   atFile,
				wantReason: "7 is no non-empty string",
			},
			{
				name:       "returns Damaged for an identity that is no object",
				give:       entryText(t, map[string]any{"identity": "equal"}),
				wantPath:   atIdentity,
				wantReason: `"equal" is no object`,
			},
			{
				name:       "returns Damaged for a null identity",
				give:       entryText(t, map[string]any{"identity": nil}),
				wantPath:   atIdentity,
				wantReason: "null is no object",
			},
			{
				name:       "returns Damaged for an identity that is an array, without its text",
				give:       entryText(t, map[string]any{"identity": []any{"equal"}}),
				wantPath:   atIdentity,
				wantReason: "the value is no object",
			},
			{
				name:       "returns Damaged for choices that are no string",
				give:       entryText(t, map[string]any{"choices": 5}),
				wantPath:   atChoices,
				wantReason: "5 is no string",
			},
			{
				name:       "returns Damaged for choices without a token version",
				give:       entryText(t, map[string]any{"choices": "AAc"}),
				wantPath:   atChoices,
				wantReason: `"AAc" states no token version`,
			},
			{
				name:       "returns Damaged for a token version with a leading zero",
				give:       entryText(t, map[string]any{"choices": "prop01:AAc"}),
				wantPath:   atChoices,
				wantReason: `"prop01:AAc" states no token version`,
			},
			{
				name:       "returns Damaged for a token not in canonical form",
				give:       entryText(t, map[string]any{"choices": "prop1:AAc="}),
				wantPath:   atChoices,
				wantReason: "the choices are no token",
			},
			{
				name:       "returns Damaged for a counterexample that is no array, without its text",
				give:       entryText(t, map[string]any{"counterexample": map[string]any{"label": "x"}}),
				wantPath:   atCounterexample,
				wantReason: "the value is no array",
			},
			{
				name:       "returns Damaged for a null counterexample",
				give:       entryText(t, map[string]any{"counterexample": nil}),
				wantPath:   atCounterexample,
				wantReason: "null is no array",
			},
			{
				name:       "returns Damaged for a draw that is no object",
				give:       withDraws(t, "x"),
				wantPath:   atFirstDraw,
				wantReason: `"x" is no object`,
			},
			{
				name:       "returns Damaged for a draw without a label",
				give:       withDraws(t, map[string]any{"value": 1}),
				wantPath:   atFirstDraw,
				wantReason: "the draw states no string label",
			},
			{
				name:       "returns Damaged for a label that is no string",
				give:       withDraws(t, map[string]any{"label": 5}),
				wantPath:   atFirstDraw,
				wantReason: "the draw states no string label",
			},
			{
				name:       "returns Damaged for a draw with another key",
				give:       withDraws(t, map[string]any{"label": "x", "note": "y"}),
				wantPath:   atFirstDraw,
				wantReason: "the draw states a key other than label and value",
			},
			{
				name:       "returns Damaged at index 1 for a second draw that is an array",
				give:       withDraws(t, map[string]any{"label": "x"}, []any{"y"}),
				wantPath:   fault.Path{fault.Field("counterexample"), fault.Index(1)},
				wantReason: "the value is no object",
			},
			{
				name:       "returns Damaged for a day the calendar does not have",
				give:       entryText(t, map[string]any{"found": "2026-02-30"}),
				wantPath:   atFound,
				wantReason: `"2026-02-30" is no date`,
			},
			{
				name:       "returns Damaged for a date in words",
				give:       entryText(t, map[string]any{"found": "1 October 2026"}),
				wantPath:   atFound,
				wantReason: `"1 October 2026" is no date`,
			},
			{
				name:       "returns Damaged for a date as a number",
				give:       entryText(t, map[string]any{"found": 20261001}),
				wantPath:   atFound,
				wantReason: "20261001 is no date",
			},
			{
				name:       "returns Damaged for year 0",
				give:       entryText(t, map[string]any{"found": "0000-01-01"}),
				wantPath:   atFound,
				wantReason: `"0000-01-01" is no date`,
			},
		}
		for _, tt := range damaged {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e, verdict, err := store.Read([]byte(tt.give), contract)
				assert.Equal(t, verdict, store.Damaged, "the verdict")
				assert.ErrorIs(t, err, store.ErrDamaged, "the damage")
				f := assert.ErrorAs[*fault.Error](t, err, "a fault")
				assert.Equal(t, f.Path, tt.wantPath, "the field at fault")
				assert.Equal(t, f.Reason, tt.wantReason, "what the file misstates")
				assert.Equal(t, e.Property, "", "no entry")
			})
		}

		t.Run("returns Damaged caused by the decoder's error for text that is not JSON", func(t *testing.T) {
			t.Parallel()
			_, _, err := store.Read([]byte("store: 1"), contract)
			cause := assert.ErrorAs[*json.SyntaxError](t, err, "the decoder's error")
			assert.Equal(t, cause.Offset, int64(1), "the decoder fails after reading the first byte")
		})

		t.Run("returns Damaged caused by the token's fault for choices that are no token", func(t *testing.T) {
			t.Parallel()
			_, _, err := store.Read([]byte(entryText(t, map[string]any{"choices": "prop1:AAc="})), contract)
			assert.ErrorIs(t, err, token.ErrInvalid, "the token's fault")
		})
	})
}

// TestReadAllocs checks the ceiling of Read on the pinned entry.
func TestReadAllocs(t *testing.T) {
	text, err := json.Marshal(pinned())
	assert.NoError(t, err, "the pinned entry is JSON")
	assert.MaxAllocs(t, func() { _, _, _ = store.Read(text, contract) }, readAllocs, "Read allocates its JSON")
}

// BenchmarkRead measures Read on the pinned entry.
func BenchmarkRead(b *testing.B) {
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
