// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/prop"
)

// The allocations of the derivations, measured.
const (
	// ofAllocs are the allocations of Of of a type that a call read before:
	// none, because Of reads each type once.
	ofAllocs = 0
	// shapeOfAllocs are the allocations of ShapeOf of an order: its read with
	// the path of each part, the read of the shape file, and the text. The
	// count varies by one between runs of the test binary, and this is the
	// higher.
	shapeOfAllocs = 387
	// ofShapeAllocs are the allocations of OfShape of a record of an int and a
	// bool: the read of the shape file, the parse of its JSON, and the
	// converters with the path of each field.
	ofShapeAllocs = 160
)

// recordOfTwo is the shape file of a record of an int and a bool.
const recordOfTwo = `{"shape":"record","fields":[["id",{"shape":"int","width":8,"signed":false}],` +
	`["ok",{"shape":"bool"}]]}`

// The types whose generators the derivation tests derive.
type (
	// digit is an integer whose generator Register states.
	digit int8
	// tally is a list of digits.
	tally struct {
		Digits []digit `json:"digits"`
	}
)

// init registers the generator of digit in the test process's registry,
// before any property runs.
func init() {
	prop.Register(prop.Integer[digit](0, 9))
}

// TestDerive checks the entry points of the derivation: the generator that
// Of returns, the shape file that ShapeOf writes, the generator that
// OfShape reads, and the operation of each of their faults.
func TestDerive(t *testing.T) {
	t.Parallel()

	t.Run("Of", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the generator that Register states for a type", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, decodedBy(prop.Of[digit](), integer(7)), digit(7), "the registered generator's value")
		})

		t.Run("returns the generator of a type that contains a type with a registered generator", func(t *testing.T) {
			t.Parallel()
			got := decodedBy(prop.Of[tally](), integer(1), integer(3), integer(0))
			assert.Equal(t, got, tally{Digits: []digit{3}}, "one digit of the registered generator")
		})

		t.Run("returns the generator of a type's shape", func(t *testing.T) {
			t.Parallel()
			got := decodedBy(prop.Of[order](), integer(7), integer(0), integer(0))
			assert.Equal(t, got, order{ID: 7}, "an order of no line and no note")
		})

		t.Run("panics for a type that the reader refuses, and states the fault", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.Of[struct{ C chan int }]() }, "a channel")
			assert.Equal(t, got, any("prop: Of[struct { C chan int }]: struct { C chan int }.C: "+
				"chan int is no type that a shape states"), "the panic names the type and the field")
		})

		t.Run("panics for a constraint that the definition refuses, and states the fault", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() {
				prop.Of[struct {
					V int32 `prop:"min=x"`
				}]()
			}, "a bound that is no integer")
			assert.Equal(t, got, any("prop: Of[struct { V int32 \"prop:\\\"min=x\\\"\" }]: "+
				`fields[0][1].min: "x" is no integer`), "the panic names the bound")
		})
	})

	t.Run("ShapeOf", func(t *testing.T) {
		t.Parallel()

		files := []struct {
			name string
			give func() (string, error)
			want string
		}{
			{name: "writes the shape file of a struct", give: prop.ShapeOf[order], want: "order.shape.json"},
			{
				name: "writes the shape file of a type that refers to itself",
				give: prop.ShapeOf[tree],
				want: "tree.shape.json",
			},
			{
				name: "writes the shape file of a registered enum",
				give: prop.ShapeOf[payment],
				want: "payment.shape.json",
			},
		}
		for _, tt := range files {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				text, err := tt.give()
				assert.NoError(t, err, "the type reads")
				golden.Match(t, tt.want, []byte(text), golden.ShouldUpdate())
			})
		}

		sources := []struct {
			name string
			give func() (string, error)
			want string
		}{
			{
				name: "states the language and the qualified name of a named type as the source",
				give: prop.ShapeOf[order],
				want: `{"language":"go","type":"go.dokimi.dev/assert/prop_test.order"}`,
			},
			{
				name: "states the name of a predeclared type as the source",
				give: prop.ShapeOf[bool],
				want: `{"language":"go","type":"bool"}`,
			},
			{
				name: "states the literal of a type without a name as the source",
				give: prop.ShapeOf[[]int8],
				want: `{"language":"go","type":"[]int8"}`,
			},
		}
		for _, tt := range sources {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				text, err := tt.give()
				assert.NoError(t, err, "the type reads")
				file := jsonTree(t, text).(map[string]any)
				assert.Equal(t, file["source"], jsonTree(t, tt.want), "the source of the shape file")
			})
		}

		faults := []struct {
			name string
			give func() (string, error)
			want fault.Error
		}{
			{
				name: "returns a fault at the part of a type that the reader refuses",
				give: prop.ShapeOf[struct{ C chan int }],
				want: fault.Error{
					Op:     shapeOfOp,
					Path:   fault.Path{fault.Field("struct { C chan int }"), fault.Field("C")},
					Reason: "chan int is no type that a shape states",
				},
			},
			{
				name: "returns a fault at the part of a type that a registered generator generates",
				give: prop.ShapeOf[tally],
				want: fault.Error{
					Op:     shapeOfOp,
					Path:   fault.Path{fault.Field("tally"), fault.Field("Digits"), fault.Element()},
					Reason: "the type has a registered generator, which states no shape",
				},
			},
			{
				name: "returns a fault for a type that a registered generator generates",
				give: prop.ShapeOf[digit],
				want: fault.Error{
					Op:     shapeOfOp,
					Path:   fault.Path{fault.Field("digit")},
					Reason: "the type has a registered generator, which states no shape",
				},
			},
			{
				name: "returns a fault in the shape file for a constraint that the definition refuses",
				give: prop.ShapeOf[struct {
					V int32 `prop:"min=x"`
				}],
				want: fault.Error{
					Op:     shapeOfOp,
					Path:   fault.Path{fault.Field("fields"), fault.Index(0), fault.Index(1), fault.Field("min")},
					Kind:   prop.ErrShape,
					Reason: `"x" is no integer`,
				},
			},
		}
		for _, tt := range faults {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := tt.give()
				expectFault(t, err, tt.want)
			})
		}
	})

	t.Run("OfShape", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the generator of a shape file that ShapeOf wrote", func(t *testing.T) {
			t.Parallel()
			text, err := prop.ShapeOf[order]()
			assert.NoError(t, err, "the type reads")
			g, err := prop.OfShape(text)
			assert.NoError(t, err, "the shape file reads")
			want := map[string]any{"id": uint32(7), "lines": []any(nil), "note": nil}
			assert.Equal(t, decodedBy(g, integer(7), integer(0), integer(0)), any(want), "the record of an order")
		})

		t.Run("returns a fault in the shape file for text that the definition refuses", func(t *testing.T) {
			t.Parallel()
			_, err := prop.OfShape(`{"shape":"widget"}`)
			expectFault(t, err, fault.Error{
				Op:     ofShapeOp,
				Path:   fault.Path{fault.Field("shape")},
				Kind:   prop.ErrShape,
				Reason: `"widget" is no shape of the vocabulary`,
			})
		})

		t.Run("returns a fault that errors.Is matches with ErrShape for text that is no JSON", func(t *testing.T) {
			t.Parallel()
			_, err := prop.OfShape(`{`)
			assert.ErrorIs(t, err, prop.ErrShape, "a shape file that does not read")
		})

		t.Run("returns a fault at the key of a map that no Go map accepts", func(t *testing.T) {
			t.Parallel()
			_, err := prop.OfShape(`{"shape":"map","key":{"shape":"bytes"},"of":{"shape":"bool"}}`)
			expectFault(t, err, fault.Error{
				Op:     ofShapeOp,
				Path:   fault.Path{fault.Field("key")},
				Reason: "the shape decodes to values that no Go map accepts as keys",
			})
		})
	})
}

// TestDeriveAllocs checks the allocation ceilings of ShapeOf and
// OfShape, and that Of of a type that a call read before allocates nothing.
func TestDeriveAllocs(t *testing.T) {
	_ = prop.Of[order]()
	assert.MaxAllocs(t, func() { _ = prop.Of[order]() }, ofAllocs, "Of reads a type once")
	assert.MaxAllocs(t, func() { _, _ = prop.ShapeOf[order]() }, shapeOfAllocs, "ShapeOf reads the type")
	assert.MaxAllocs(t, func() { _, _ = prop.OfShape(recordOfTwo) }, ofShapeAllocs, "OfShape reads the shape file")
}

// BenchmarkDerive measures Of of a type that a call read before, ShapeOf of
// an order, and OfShape of a record of an int and a bool.
func BenchmarkDerive(b *testing.B) {
	b.Run("Of", func(b *testing.B) {
		_ = prop.Of[order]()
		var got prop.Generator[order]
		c := bench.Start(b).MaxAllocs(ofAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Of[order]()
		}
		assert.Equal(b, decodedBy(got, integer(7), integer(0), integer(0)), order{ID: 7}, "an order of no line")
	})

	b.Run("ShapeOf", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(shapeOfAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = prop.ShapeOf[order]()
		}
		assert.Contains(b, got, `"source"`, "the shape file names its source")
	})

	b.Run("OfShape", func(b *testing.B) {
		var got prop.Generator[any]
		c := bench.Start(b).MaxAllocs(ofShapeAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = prop.OfShape(recordOfTwo)
		}
		assert.Equal(b, decodedBy(got, integer(7), integer(1)), any(map[string]any{"id": uint8(7), "ok": true}),
			"the record's Go value")
	})
}
