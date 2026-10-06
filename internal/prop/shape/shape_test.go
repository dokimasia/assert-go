// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"io"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/shape"
)

// The allocations of the reader, measured.
const (
	// readAllocs are the allocations of Read of a record of an int and a
	// bool: the parsed document, the reader, and the generators.
	readAllocs = 97
	// readWithAllocs are the allocations of ReadWith of that record and one
	// external generator: those of Read, and the growth of the reader's
	// generators.
	readWithAllocs = 98
	// shapesAllocs are the allocations of Shapes: its list of ids.
	shapesAllocs = 1
)

// infinite is the reason of a definition without a finite value.
const infinite = "the definition has no finite value, because it refers back without an optional, a list, " +
	"a set, a map or an enum that can exit"

// FuzzRead checks that Read returns a generator or a fault of the kind
// ErrShape for any bytes, and does not panic. A case of seed 9 draws from the
// generator of a document that Read accepts without failing, and a value
// that it draws runs back, as runsBack checks.
func FuzzRead(f *testing.F) {
	f.Add([]byte(recordShape))
	f.Add([]byte(`{"shape":"set","of":` + uint8Shape + `,"max_size":4}`))
	f.Add([]byte(`{"shape":"ref","name":"t","definitions":{"t":{"shape":"optional",` +
		`"of":{"shape":"list","of":{"shape":"ref","name":"t"},"max_size":2}}}}`))
	f.Add([]byte(`{"shape":"string","pattern":"[a-c]{2,4}"}`))
	f.Add([]byte(`{"shape":"map","key":{"shape":"string","max_size":2},"of":{"shape":"float","width":64}}`))
	f.Add([]byte(`{"shape":"int","width":8,"signed":true,"mx":1}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, document []byte) {
		g, err := shape.Read(document)
		if err != nil {
			assert.ErrorIs(t, err, shape.ErrShape, "a fault of the kind ErrShape")
			return
		}
		var v any
		e := engine.Generate(func(c *engine.Case) { v = engine.Draw(c, g, drawn) }, 9, 0, nil)
		assert.NotEqual(t, e.Status, engine.CaseFailed, "the generator decodes a case without failing")
		if e.Status == engine.CasePassed {
			runsBack(t, string(document), g, v)
		}
	})
}

// TestShape checks the vocabulary, the documents that fail to read and
// where each fault is, and the generator of a document's root.
func TestShape(t *testing.T) {
	t.Parallel()

	t.Run("Shapes", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the 27 shapes of the definition, sorted", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, shape.Shapes(), []string{
				"bool", "bytes", "char", "date", "decimal", "duration", "enum", "fixed-list", "float", "instant",
				"int", "ip-address", "list", "literal", "local-date-time", "map", "offset", "optional", "record",
				"ref", "set", "string", "time-of-day", "uuid", "wall-time", "zone", "zoned-date-time",
			}, "the vocabulary")
		})
	})

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		definitions := fault.Field("definitions")
		faults := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{name: "returns an error for text that is no JSON", give: `{`, wantReason: "the document is no JSON"},
			{
				name:       "returns an error for two JSON values",
				give:       `{} {}`,
				wantReason: "the document states more than its JSON value",
			},
			{
				name:       "returns an error for a closing bracket after the JSON value",
				give:       `{"shape":"bool"}]`,
				wantReason: "the document states more than its JSON value",
			},
			{
				name:       "returns an error for text after the JSON value",
				give:       `{"shape":"bool"}}garbage`,
				wantReason: "the document states more than its JSON value",
			},
			{name: "returns an error for a document that is no object", give: `[]`, wantReason: "[] is no shape"},
			{
				name:       "returns an error at the id of an unknown shape",
				give:       `{"shape":"widget"}`,
				wantPath:   fault.Path{fault.Field("shape")},
				wantReason: `"widget" is no shape of the vocabulary`,
			},
			{
				name:       "returns an error at the id of a shape that states none",
				give:       `{}`,
				wantPath:   fault.Path{fault.Field("shape")},
				wantReason: "null is no shape of the vocabulary",
			},
			{
				name:       "returns an error for a missing key",
				give:       `{"shape":"int","width":8}`,
				wantReason: "the int shape needs the key signed",
			},
			{
				name:       "returns an error at a key the shape does not take",
				give:       `{"shape":"int","width":8,"signed":true,"mx":1}`,
				wantPath:   fault.Path{fault.Field("mx")},
				wantReason: "the key does not apply to the int shape",
			},
			{
				name:       "returns an error at the element shape of a list that is no shape",
				give:       `{"shape":"list","of":3}`,
				wantPath:   fault.Path{fault.Field("of")},
				wantReason: "3 is no shape",
			},
			{
				name:       "returns an error at the value shape of an optional that is no shape",
				give:       `{"shape":"optional","of":3}`,
				wantPath:   fault.Path{fault.Field("of")},
				wantReason: "3 is no shape",
			},
			{
				name:       "returns an error at a map key that is no shape",
				give:       `{"shape":"map","key":3,"of":{"shape":"bool"}}`,
				wantPath:   fault.Path{fault.Field("key")},
				wantReason: "3 is no shape",
			},
			{
				name:       "returns an error at a map value that is no shape",
				give:       `{"shape":"map","key":{"shape":"bool"},"of":3}`,
				wantPath:   fault.Path{fault.Field("of")},
				wantReason: "3 is no shape",
			},
			{
				name:       "returns an error at definitions that are no map",
				give:       `{"shape":"bool","definitions":[]}`,
				wantPath:   fault.Path{definitions},
				wantReason: "[] is no map of names to shapes",
			},
			{
				name:       "returns an error at a definition that is no shape",
				give:       `{"shape":"bool","definitions":{"id":3}}`,
				wantPath:   fault.Path{definitions, fault.Key("id")},
				wantReason: "3 is no shape",
			},
			{
				name:       "returns an error inside a definition that does not read",
				give:       `{"shape":"bool","definitions":{"id":{"shape":"widget"}}}`,
				wantPath:   fault.Path{definitions, fault.Key("id"), fault.Field("shape")},
				wantReason: `"widget" is no shape of the vocabulary`,
			},
			{
				name:       "returns an error at a source without a type",
				give:       `{"shape":"bool","source":{"language":"go"}}`,
				wantPath:   fault.Path{fault.Field("source")},
				wantReason: `{"language":"go"} is no language and type`,
			},
			{
				name:       "returns an error at a source that is no object",
				give:       `{"shape":"bool","source":"go"}`,
				wantPath:   fault.Path{fault.Field("source")},
				wantReason: `"go" is no language and type`,
			},
			{
				name:       "returns an error at a source below the root",
				give:       `{"shape":"list","of":{"shape":"bool","source":{}}}`,
				wantPath:   fault.Path{fault.Field("of"), fault.Field("source")},
				wantReason: "the key does not apply to the bool shape",
			},
			{
				name:       "returns an error for a ref to no definition",
				give:       `{"shape":"ref","name":"missing"}`,
				wantReason: `a ref names "missing", which is no definition`,
			},
			{
				name:       "returns an error for a ref whose name is no string",
				give:       `{"shape":"ref","name":3}`,
				wantReason: `a ref names "3", which is no definition`,
			},
			{
				name: "returns an error at a record that contains itself",
				give: `{"shape":"ref","name":"loop","definitions":{"loop":{"shape":"record","fields":[` +
					`["next",{"shape":"ref","name":"loop"}]]}}}`,
				wantPath:   fault.Path{definitions, fault.Key("loop")},
				wantReason: infinite,
			},
			{
				name: "returns an error at a list that must contain itself",
				give: `{"shape":"ref","name":"t","definitions":{"t":{"shape":"list","min_size":1,` +
					`"of":{"shape":"ref","name":"t"}}}}`,
				wantPath:   fault.Path{definitions, fault.Key("t")},
				wantReason: infinite,
			},
			{
				name: "returns an error at an enum whose every variant refers back",
				give: `{"shape":"ref","name":"e","definitions":{"e":{"shape":"enum","variants":[` +
					`["only",{"shape":"ref","name":"e"}]]}}}`,
				wantPath:   fault.Path{definitions, fault.Key("e")},
				wantReason: infinite,
			},
			{
				name: "returns an error at the first of two records that contain each other",
				give: `{"shape":"ref","name":"a","definitions":{` +
					`"a":{"shape":"record","fields":[["b",{"shape":"ref","name":"b"}]]},` +
					`"b":{"shape":"record","fields":[["a",{"shape":"ref","name":"a"}]]}}}`,
				wantPath:   fault.Path{definitions, fault.Key("a")},
				wantReason: infinite,
			},
			{
				name: "returns an error at the one definition of a cycle that no container can exit",
				give: `{"shape":"ref","name":"a","definitions":{` +
					`"a":{"shape":"record","fields":[["b",{"shape":"ref","name":"b"}]]},` +
					`"b":{"shape":"record","fields":[["c",{"shape":"ref","name":"c"}]]},` +
					`"c":{"shape":"optional","of":{"shape":"record","fields":[["a",{"shape":"ref","name":"a"}],` +
					`["b",{"shape":"ref","name":"b"}]]}},"d":{"shape":"record","fields":[["d",{"shape":"ref","name":"d"}]]}}}`,
				wantPath:   fault.Path{definitions, fault.Key("d")},
				wantReason: infinite,
			},
		}
		for _, tt := range faults {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := shape.Read([]byte(tt.give))
				isFault(t, err, shape.ErrShape, tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns an error caused by the decoder's error for text that ends inside a value", func(t *testing.T) {
			t.Parallel()
			_, err := shape.Read([]byte(`{`))
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF, "the decoder's error")
		})

		t.Run("returns a root that states its source", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, `{"shape":"int","width":8,"signed":false,`+
				`"source":{"language":"go","type":"example.com/x.Qty"}}`, n(7))
			assert.Equal(t, got, any(uint64(7)), "the source changes nothing")
		})

		t.Run("returns a root of a null source", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, `{"shape":"bool","source":null}`, n(1))
			assert.Equal(t, got, any(true), "a null source states none")
		})

		t.Run("returns the generator of an int's root, whose draw the explain phase steps", func(t *testing.T) {
			t.Parallel()
			g := read(t, `{"shape":"int","width":32,"signed":true}`)
			got := engine.Run(func(c *engine.Case) {
				if engine.Draw(c, g, drawn).(int64) >= 5 {
					c.Report(assert.Failure{Assertion: "big"}, false)
				}
			}, engine.Settings{
				Seed: 7, Cases: engine.DefaultCases, MaxChoices: engine.MaxChoices,
				Shrink: engine.DefaultShrink, Explain: true,
			})
			assert.Equal(t, got.Explanation, []engine.Explained{
				{Label: drawn, Value: int64(5), Relevance: engine.ValueMatters, NearestPassing: int64(4)},
			}, "the nearest passing value of a draw from integer")
		})
	})

	t.Run("ReadWith", func(t *testing.T) {
		t.Parallel()

		digits := map[string]engine.Generator[any]{"digit": engine.Erase(engine.Integer(0, 9))}
		listed := []byte(`{"shape":"list","of":{"shape":"ref","name":"digit"}}`)

		t.Run("returns a shape whose ref to an external name is its generator", func(t *testing.T) {
			t.Parallel()
			g, err := shape.ReadWith(listed, digits)
			assert.NoError(t, err, "the external name is known")
			var got any
			engine.Replay(
				func(c *engine.Case) { got = engine.Draw(c, g, drawn) },
				[]choice.Choice{n(1), n(7), n(0)},
				nil,
			)
			assert.Equal(t, got, any([]any{7}), "the external generator's value")
			choices, err := engine.Invert(g, any([]any{7}))
			assert.NoError(t, err, "the list runs back through the external generator")
			assert.Length(t, choices, 3, "a flag, the digit and a stop")
		})

		t.Run("returns an error for a ref that names neither a definition nor an external name", func(t *testing.T) {
			t.Parallel()
			_, err := shape.ReadWith(listed, nil)
			isFault(t, err, shape.ErrShape, nil, `a ref names "digit", which is no definition`)
		})
	})
}

// TestShapeAllocs checks the allocation ceilings of Read, ReadWith and
// Shapes.
func TestShapeAllocs(t *testing.T) {
	text := []byte(recordShape)
	assert.MaxAllocs(t, func() { _, _ = shape.Read(text) }, readAllocs,
		"Read allocates the document and the generators")
	externals := map[string]engine.Generator[any]{"digit": engine.Erase(engine.Integer(0, 9))}
	assert.MaxAllocs(t, func() { _, _ = shape.ReadWith(text, externals) }, readWithAllocs,
		"ReadWith allocates what Read does and the external generators")
	assert.MaxAllocs(t, func() { _ = shape.Shapes() }, shapesAllocs, "Shapes allocates its list")
}

// BenchmarkShape measures Read and ReadWith of a record of an int and a
// bool, and Shapes.
func BenchmarkShape(b *testing.B) {
	b.Run("Read", func(b *testing.B) {
		text := []byte(recordShape)
		var got engine.Generator[any]
		c := bench.Start(b).MaxAllocs(readAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = shape.Read(text)
		}
		assert.Equal(b, got.ID(), "record", "the root's id")
	})

	b.Run("ReadWith", func(b *testing.B) {
		text := []byte(recordShape)
		externals := map[string]engine.Generator[any]{"digit": engine.Erase(engine.Integer(0, 9))}
		var got engine.Generator[any]
		c := bench.Start(b).MaxAllocs(readWithAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = shape.ReadWith(text, externals)
		}
		assert.Equal(b, got.ID(), "record", "the root's id")
	})

	b.Run("Shapes", func(b *testing.B) {
		var got []string
		c := bench.Start(b).MaxAllocs(shapesAllocs)
		defer c.End()
		for c.Loop() {
			got = shape.Shapes()
		}
		assert.Length(b, got, 27, "the vocabulary")
	})
}
