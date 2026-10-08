// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"math"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/pattern"
	"go.dokimi.dev/assert/internal/prop/shape"
)

// Shapes that the structural tests decode.
const (
	// int8Shape is a signed byte.
	int8Shape = `{"shape":"int","width":8,"signed":true}`
	// either is an enum of a variant without a payload and one with a byte.
	either = `{"shape":"enum","variants":[["none",null],["some",` + uint8Shape + `]]}`
)

// structuralTrips are a shape of each structural kind, whose generated
// values each run back to choices that decode to them.
var structuralTrips = []string{
	`{"shape":"bool"}`,
	int8Shape,
	`{"shape":"float","width":32,"allow_nan":true,"allow_infinity":true}`,
	`{"shape":"char","alphabet":"xyz"}`,
	`{"shape":"string","max_size":4}`,
	`{"shape":"string","pattern":"[a-c]{2}x?"}`,
	`{"shape":"bytes","max_size":3}`,
	`{"shape":"list","of":` + uint8Shape + `,"max_size":4}`,
	`{"shape":"fixed-list","of":{"shape":"bool"},"size":3}`,
	`{"shape":"set","of":` + uint8Shape + `,"max_size":4}`,
	`{"shape":"map","key":{"shape":"string","max_size":2},"of":{"shape":"bool"},"max_size":3}`,
	`{"shape":"optional","of":` + uint8Shape + `}`,
	`{"shape":"optional","of":{"shape":"list","of":{"shape":"bool"},"max_size":2}}`,
	recordShape,
	`{"shape":"enum","variants":[["none",null],["some",` + uint8Shape + `],["maybe",{"shape":"optional","of":{"shape":"bool"}}]]}`,
	`{"shape":"literal","values":[{"type":"string","value":"paid"},{"type":"int","value":3}]}`,
}

// TestStructural checks the structural shapes: the value that each decodes
// from stated choices, the choices that each value runs back to, and each
// shape and each value that a shape refuses, with where the fault is.
func TestStructural(t *testing.T) {
	t.Parallel()

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		values := []struct {
			name  string
			shape string
			give  []choice.Choice
			want  any
		}{
			{name: "returns true for 1 of a bool", shape: `{"shape":"bool"}`, give: []choice.Choice{n(1)}, want: true},
			{name: "returns false for the target of a bool", shape: `{"shape":"bool"}`, want: false},
			{
				name:  "returns the least signed byte",
				shape: int8Shape,
				give:  []choice.Choice{n(-128)},
				want:  int64(-128),
			},
			{
				name:  "returns the target for a signed byte past the range",
				shape: int8Shape,
				give:  []choice.Choice{n(128)},
				want:  int64(0),
			},
			{
				name:  "returns the largest unsigned byte",
				shape: uint8Shape,
				give:  []choice.Choice{n(255)},
				want:  uint64(255),
			},
			{
				name:  "returns the target for a negative unsigned byte",
				shape: uint8Shape,
				give:  []choice.Choice{n(-1)},
				want:  uint64(0),
			},
			{
				name:  "returns the largest value of an int's bounds",
				shape: `{"shape":"int","width":8,"signed":true,"min":1,"max":99}`,
				give:  []choice.Choice{n(99)},
				want:  int64(99),
			},
			{
				name:  "returns the target of an int's bounds for a value past them",
				shape: `{"shape":"int","width":8,"signed":true,"min":1,"max":99}`,
				give:  []choice.Choice{n(100)},
				want:  int64(1),
			},
			{
				name:  "returns the largest unsigned 64-bit int",
				shape: `{"shape":"int","width":64,"signed":false}`,
				give:  []choice.Choice{allOnes},
				want:  uint64(math.MaxUint64),
			},
			{
				name:  "returns the target of a finite float for an infinity",
				shape: `{"shape":"float","width":64}`,
				give:  []choice.Choice{float(math.Inf(1))},
				want:  0.0,
			},
			{
				name:  "returns an infinity that allow_infinity admits",
				shape: `{"shape":"float","width":64,"allow_infinity":true}`,
				give:  []choice.Choice{float(math.Inf(1))},
				want:  math.Inf(1),
			},
			{
				name:  "returns the target of a binary32 float for 0.1",
				shape: `{"shape":"float","width":32}`,
				give:  []choice.Choice{float(0.1)},
				want:  float32(0),
			},
			{
				name:  "returns 0.5 of a binary32 float",
				shape: `{"shape":"float","width":32}`,
				give:  []choice.Choice{float(0.5)},
				want:  float32(0.5),
			},
			{
				name:  "returns a float of stated bounds and names",
				shape: `{"shape":"float","width":64,"min":"-Inf","max":2,"allow_nan":true}`,
				give:  []choice.Choice{float(math.Inf(-1))},
				want:  math.Inf(-1),
			},
			{
				name:  "returns the infinity of a maximum that names it",
				shape: `{"shape":"float","width":64,"min":0,"max":"Inf","allow_infinity":true}`,
				give:  []choice.Choice{float(math.Inf(1))},
				want:  math.Inf(1),
			},
			{
				name:  "returns a char of the default alphabet",
				shape: `{"shape":"char"}`,
				give:  []choice.Choice{seq(10)},
				want:  "a",
			},
			{
				name:  "returns a char of a stated alphabet",
				shape: `{"shape":"char","alphabet":"xyz"}`,
				give:  []choice.Choice{seq(2)},
				want:  "z",
			},
			{
				name:  "returns a string of the minimum length",
				shape: `{"shape":"string","min_size":2,"max_size":3}`,
				give:  []choice.Choice{seq(1)},
				want:  "10",
			},
			{
				name:  "returns a string of a stated alphabet",
				shape: `{"shape":"string","alphabet":"ab"}`,
				give:  []choice.Choice{seq(1, 0)},
				want:  "ba",
			},
			{
				name:  "returns a string that a pattern matches",
				shape: `{"shape":"string","pattern":"a|b"}`,
				give:  []choice.Choice{n(1)},
				want:  "b",
			},
			{
				name:  "returns bytes",
				shape: `{"shape":"bytes"}`,
				give:  []choice.Choice{seq(0, 255)},
				want:  []byte{0, 255},
			},
			{
				name:  "returns a list's elements",
				shape: `{"shape":"list","of":` + uint8Shape + `}`,
				give:  []choice.Choice{n(1), n(7), n(1), n(8), n(0)},
				want:  []any{uint64(7), uint64(8)},
			},
			{
				name:  "returns a fixed-list's elements",
				shape: `{"shape":"fixed-list","of":` + uint8Shape + `,"size":2}`,
				give:  []choice.Choice{n(1), n(7), n(1), n(8), n(0)},
				want:  []any{uint64(7), uint64(8)},
			},
			{
				name:  "returns a set without a repeated element",
				shape: `{"shape":"set","of":` + uint8Shape + `}`,
				give:  []choice.Choice{n(1), n(7), n(1), n(7), n(0)},
				want:  []any{uint64(7)},
			},
			{
				name:  "returns a map's entries",
				shape: `{"shape":"map","key":` + uint8Shape + `,"of":{"shape":"bool"}}`,
				give:  []choice.Choice{n(1), n(3), n(1), n(0)},
				want:  literal.Pairs{Entries: []literal.Entry{{Key: uint64(3), Value: true}}},
			},
			{
				name:  "returns nil for an absent optional",
				shape: `{"shape":"optional","of":` + uint8Shape + `}`,
				want:  nil,
			},
			{
				name:  "returns the value of a present optional",
				shape: `{"shape":"optional","of":` + uint8Shape + `}`,
				give:  []choice.Choice{n(1), n(5)},
				want:  uint64(5),
			},
			{
				name:  "returns a present empty list as a non-nil slice, which no optional reads as absent",
				shape: `{"shape":"optional","of":{"shape":"list","of":{"shape":"bool"}}}`,
				give:  []choice.Choice{n(1), n(0)},
				want:  []any{},
			},
			{
				name:  "returns a record's fields in order",
				shape: recordShape,
				give:  []choice.Choice{n(4), n(1)},
				want:  record("id", uint64(4), "ok", true),
			},
			{
				name:  "returns a variant without a payload",
				shape: either,
				want:  literal.Variant{Name: "none"},
			},
			{
				name:  "returns a variant with its payload",
				shape: either,
				give:  []choice.Choice{n(1), n(9)},
				want:  literal.Variant{Name: "some", Payload: uint64(9), HasPayload: true},
			},
			{
				name:  "returns one of a literal's values",
				shape: `{"shape":"literal","values":[{"type":"string","value":"paid"},{"type":"string","value":"shipped"}]}`,
				give:  []choice.Choice{n(1)},
				want:  "shipped",
			},
		}
		for _, tt := range values {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, _ := decode(t, tt.shape, tt.give...)
				assert.Equal(t, got, tt.want, "the value")
			})
		}

		patterned := `{"shape":"string","pattern":"a"`
		faults := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns an error at a width outside the widths",
				give:       `{"shape":"int","width":12,"signed":true}`,
				wantPath:   fault.Path{fault.Field("width")},
				wantReason: "12 is none of the widths 8, 16, 32, 64 and 128",
			},
			{
				name:       "returns an error at signedness that is no boolean",
				give:       `{"shape":"int","width":8,"signed":"yes"}`,
				wantPath:   fault.Path{fault.Field("signed")},
				wantReason: `"yes" is no boolean`,
			},
			{
				name:       "returns an error for a bound outside the width",
				give:       `{"shape":"int","width":8,"signed":true,"min":200}`,
				wantReason: "the bounds [200, 127] are empty or outside [-128, 127]",
			},
			{
				name:       "returns an error at the bound of a record's field that is no integer",
				give:       `{"shape":"record","fields":[["id",{"shape":"int","width":8,"signed":false,"max":"x"}]]}`,
				wantPath:   fault.Path{fault.Field("fields"), fault.Index(0), fault.Index(1), fault.Field("max")},
				wantReason: `"x" is no integer`,
			},
			{
				name:       "returns an error at the name of a field named twice",
				give:       `{"shape":"record","fields":[["id",{"shape":"bool"}],["id",{"shape":"bool"}]]}`,
				wantPath:   fault.Path{fault.Field("fields"), fault.Index(1), fault.Index(0)},
				wantReason: `the name "id" is empty or named twice`,
			},
			{
				name:       "returns an error at the name of a field of an empty name",
				give:       `{"shape":"record","fields":[["",{"shape":"bool"}]]}`,
				wantPath:   fault.Path{fault.Field("fields"), fault.Index(0), fault.Index(0)},
				wantReason: `the name "" is empty or named twice`,
			},
			{
				name:       "returns an error at the fields of a record of no field",
				give:       `{"shape":"record","fields":[]}`,
				wantPath:   fault.Path{fault.Field("fields")},
				wantReason: "[] is no list of pairs",
			},
			{
				name:       "returns an error at a variant of one part",
				give:       `{"shape":"enum","variants":[["x"]]}`,
				wantPath:   fault.Path{fault.Field("variants"), fault.Index(0)},
				wantReason: `["x"] is no name and shape`,
			},
			{
				name:       "returns an error at a variant whose name is no string",
				give:       `{"shape":"enum","variants":[[1,null]]}`,
				wantPath:   fault.Path{fault.Field("variants"), fault.Index(0)},
				wantReason: "[1,null] is no name and shape",
			},
			{
				name:       "returns an error at a variant that is no list",
				give:       `{"shape":"enum","variants":["x"]}`,
				wantPath:   fault.Path{fault.Field("variants"), fault.Index(0)},
				wantReason: `"x" is no name and shape`,
			},
			{
				name:       "returns an error inside a payload that does not read",
				give:       `{"shape":"enum","variants":[["x",{"shape":"widget"}]]}`,
				wantPath:   fault.Path{fault.Field("variants"), fault.Index(0), fault.Index(1), fault.Field("shape")},
				wantReason: `"widget" is no shape of the vocabulary`,
			},
			{
				name:       "returns an error at the null size of a fixed-list",
				give:       `{"shape":"fixed-list","of":{"shape":"bool"},"size":null}`,
				wantPath:   fault.Path{fault.Field("size")},
				wantReason: "null is no size",
			},
			{
				name:       "returns an error for a pattern with a size",
				give:       patterned + `,"max_size":3}`,
				wantReason: "the shape states a pattern and an alphabet or a size",
			},
			{
				name:       "returns an error for a pattern with a null alphabet",
				give:       patterned + `,"alphabet":null}`,
				wantReason: "the shape states a pattern and an alphabet or a size",
			},
			{
				name:       "returns an error for a pattern with a minimum size",
				give:       patterned + `,"min_size":0}`,
				wantReason: "the shape states a pattern and an alphabet or a size",
			},
			{
				name:       "returns an error at a pattern that is no string",
				give:       `{"shape":"string","pattern":3}`,
				wantPath:   fault.Path{fault.Field("pattern")},
				wantReason: "3 is no pattern",
			},
			{
				name:       "returns an error at a pattern outside the portable subset",
				give:       `{"shape":"string","pattern":"a**"}`,
				wantPath:   fault.Path{fault.Field("pattern")},
				wantReason: "the pattern is outside the portable subset",
			},
			{
				name:       "returns an error at an alphabet that repeats a character",
				give:       `{"shape":"string","alphabet":"aa"}`,
				wantPath:   fault.Path{fault.Field("alphabet")},
				wantReason: "the alphabet repeats a character",
			},
			{
				name:       "returns an error at an alphabet that is no string",
				give:       `{"shape":"char","alphabet":3}`,
				wantPath:   fault.Path{fault.Field("alphabet")},
				wantReason: "3 is no string of characters",
			},
			{
				name:       "returns an error at an empty alphabet",
				give:       `{"shape":"char","alphabet":""}`,
				wantPath:   fault.Path{fault.Field("alphabet")},
				wantReason: `"" is no string of characters`,
			},
			{
				name:       "returns an error at an alphabet that is a list",
				give:       `{"shape":"string","alphabet":["a"]}`,
				wantPath:   fault.Path{fault.Field("alphabet")},
				wantReason: `["a"] is no string of characters`,
			},
			{
				name:       "returns an error at a float of width 16",
				give:       `{"shape":"float","width":16}`,
				wantPath:   fault.Path{fault.Field("width")},
				wantReason: "16 is neither 32 nor 64",
			},
			{
				name:       "returns an error at a float width beyond the int64 range",
				give:       `{"shape":"float","width":"18446744073709551648"}`,
				wantPath:   fault.Path{fault.Field("width")},
				wantReason: "18446744073709551648 is neither 32 nor 64",
			},
			{
				name:       "returns an error at a float width that is no integer",
				give:       `{"shape":"float","width":64.5}`,
				wantPath:   fault.Path{fault.Field("width")},
				wantReason: "64.5 is no integer",
			},
			{
				name:       "returns an error at allow_nan that is no boolean",
				give:       `{"shape":"float","width":64,"allow_nan":"yes"}`,
				wantPath:   fault.Path{fault.Field("allow_nan")},
				wantReason: `"yes" is no boolean`,
			},
			{
				name:       "returns an error at allow_infinity of null",
				give:       `{"shape":"float","width":64,"allow_infinity":null}`,
				wantPath:   fault.Path{fault.Field("allow_infinity")},
				wantReason: "null is no boolean",
			},
			{
				name:       "returns an error for infinities that the bounds leave out",
				give:       `{"shape":"float","width":64,"allow_infinity":true,"min":0,"max":1}`,
				wantReason: "the shape allows the infinities, and its bounds leave both out",
			},
			{
				name:       "returns an error at a float minimum that is no float",
				give:       `{"shape":"float","width":64,"min":"x"}`,
				wantPath:   fault.Path{fault.Field("min")},
				wantReason: `"x" is no float`,
			},
			{
				name:       "returns an error at a float maximum that is no float",
				give:       `{"shape":"float","width":64,"max":true}`,
				wantPath:   fault.Path{fault.Field("max")},
				wantReason: "true is no float",
			},
			{
				name:       "returns an error for float bounds of no value",
				give:       `{"shape":"float","width":64,"min":2,"max":1}`,
				wantReason: "the bounds admit no float",
			},
			{
				name:       "returns an error for a float bound of NaN",
				give:       `{"shape":"float","width":64,"min":"NaN"}`,
				wantReason: "the bounds admit no float",
			},
			{
				name:       "returns an error at the values of a literal of none",
				give:       `{"shape":"literal","values":[]}`,
				wantPath:   fault.Path{fault.Field("values")},
				wantReason: "[] is no list of literals",
			},
			{
				name:       "returns an error at a literal that does not decode",
				give:       `{"shape":"literal","values":[{"type":"widget"}]}`,
				wantPath:   fault.Path{fault.Field("values"), fault.Index(0)},
				wantReason: "the value is no typed literal",
			},
		}
		for _, tt := range faults {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := shape.Read([]byte(tt.give))
				isFault(t, err, shape.ErrShape, tt.wantPath, tt.wantReason)
			})
		}

		causes := []struct {
			name string
			give string
			want error
		}{
			{
				name: "returns an error caused by the fault of a pattern outside the portable subset",
				give: `{"shape":"string","pattern":"a**"}`,
				want: pattern.ErrOutside,
			},
			{
				name: "returns an error caused by the fault of float bounds of no value",
				give: `{"shape":"float","width":64,"min":2,"max":1}`,
				want: choice.ErrEmpty,
			},
			{
				name: "returns an error caused by the fault of a literal that does not decode",
				give: `{"shape":"literal","values":[{"type":"widget"}]}`,
				want: literal.ErrUnknownType,
			},
		}
		for _, tt := range causes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := shape.Read([]byte(tt.give))
				assert.ErrorIs(t, err, tt.want, "the cause")
			})
		}
	})

	t.Run("Invert", func(t *testing.T) {
		t.Parallel()

		t.Run("returns choices that decode to each value that a structural shape generates", func(t *testing.T) {
			t.Parallel()
			for _, text := range structuralTrips {
				roundTrips(t, text)
			}
		})

		t.Run("returns a set's elements in the order of their choices, whatever their order", func(t *testing.T) {
			t.Parallel()
			g := read(t, `{"shape":"set","of":`+uint8Shape+`}`)
			forward, err := engine.Invert(g, any([]any{113, 13}))
			assert.NoError(t, err, "the set runs back")
			backward, err := engine.Invert(g, any([]any{13, 113}))
			assert.NoError(t, err, "the set runs back in the other order")
			assert.True(t, slices.EqualFunc(forward, backward, choice.Choice.Equal), "one sequence of choices")
		})

		tests := []struct {
			name  string
			shape string
			give  any
			want  []choice.Choice
		}{
			{
				name:  "returns a set's elements in the shortlex order of their choices",
				shape: `{"shape":"set","of":` + uint8Shape + `}`,
				give:  []int{5, 2},
				want:  []choice.Choice{n(1), n(2), n(1), n(5), n(0)},
			},
			{
				name:  "returns a map's entries from a Go map",
				shape: `{"shape":"map","key":` + uint8Shape + `,"of":{"shape":"bool"}}`,
				give:  map[int]bool{4: true, 1: false},
				want:  []choice.Choice{n(1), n(1), n(0), n(1), n(4), n(1), n(0)},
			},
			{
				name:  "returns an absent optional for a nil slice",
				shape: `{"shape":"optional","of":{"shape":"list","of":{"shape":"bool"}}}`,
				give:  []bool(nil),
				want:  []choice.Choice{n(0)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := engine.Invert(read(t, tt.shape), tt.give)
				assert.NoError(t, err, "the shape produces the value")
				assert.True(t, slices.EqualFunc(got, tt.want, choice.Choice.Equal), "the choices")
			})
		}

		bytesToBools := `{"shape":"map","key":` + uint8Shape + `,"of":{"shape":"bool"}}`
		refusals := []struct {
			name       string
			shape      string
			give       any
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns an error for a list's value that is no list",
				shape:      `{"shape":"list","of":{"shape":"bool"}}`,
				give:       "x",
				wantReason: "x is no list",
			},
			{
				name:       "returns an error at the element that a list does not produce",
				shape:      `{"shape":"list","of":{"shape":"bool"}}`,
				give:       []any{true, 1},
				wantPath:   fault.Path{fault.Index(1)},
				wantReason: "1 is no bool",
			},
			{
				name:       "returns an error for a fixed-list of another size",
				shape:      `{"shape":"fixed-list","of":{"shape":"bool"},"size":2}`,
				give:       []bool{true},
				wantReason: "1 elements are outside the sizes of the collection",
			},
			{
				name:       "returns an error at the element of a set that repeats an earlier one",
				shape:      `{"shape":"set","of":` + uint8Shape + `}`,
				give:       []any{1, uint64(1)},
				wantPath:   fault.Path{fault.Index(1)},
				wantReason: "the element repeats element 0",
			},
			{
				name:       "returns an error for a map's value that is no map",
				shape:      `{"shape":"map","key":{"shape":"bool"},"of":{"shape":"bool"}}`,
				give:       3,
				wantReason: "3 is no map",
			},
			{
				name:       "returns an error at a key that a map's keys do not produce",
				shape:      bytesToBools,
				give:       map[int]bool{300: true},
				wantPath:   fault.Path{fault.Key(300)},
				wantReason: "the generator of keys produces no such key",
			},
			{
				name:       "returns an error at the key of a value that a map's values do not produce",
				shape:      bytesToBools,
				give:       map[int]any{1: 2},
				wantPath:   fault.Path{fault.Key(1)},
				wantReason: "2 is no bool",
			},
			{
				name:  "returns an error for two keys of one value",
				shape: bytesToBools,
				give: literal.Pairs{
					Entries: []literal.Entry{{Key: 1, Value: true}, {Key: uint64(1), Value: false}},
				},
				wantReason: "two keys decode to the key 1",
			},
			{
				name:       "returns an error for a map smaller than its sizes",
				shape:      `{"shape":"map","key":{"shape":"bool"},"of":{"shape":"bool"},"min_size":2}`,
				give:       map[bool]bool{true: true},
				wantReason: "1 elements are outside the sizes of the collection",
			},
			{
				name:       "returns an error for a present value that the optional's shape does not produce",
				shape:      `{"shape":"optional","of":{"shape":"bool"}}`,
				give:       3,
				wantReason: "3 is no bool",
			},
			{
				name:       "returns an error for a record's value that is no record",
				shape:      recordShape,
				give:       3,
				wantReason: "3 is no record of the fields [id ok]",
			},
			{
				name:       "returns an error for a record of other fields",
				shape:      recordShape,
				give:       record("ok", true, "id", 4),
				wantReason: "{[{ok true} {id 4}]} is no record of the fields [id ok]",
			},
			{
				name:       "returns an error for a record of fewer fields",
				shape:      recordShape,
				give:       literal.Record{Fields: []literal.Field{{Name: "id", Value: 4}}},
				wantReason: "{[{id 4}]} is no record of the fields [id ok]",
			},
			{
				name:       "returns an error at the field that a record does not produce",
				shape:      recordShape,
				give:       record("id", 400, "ok", true),
				wantPath:   fault.Path{fault.Field("id")},
				wantReason: "400 is outside [0, 255]",
			},
			{
				name:       "returns an error for an enum's value that is no variant",
				shape:      either,
				give:       "none",
				wantReason: "none is no variant of [none some]",
			},
			{
				name:       "returns an error for an unknown variant",
				shape:      either,
				give:       literal.Variant{Name: "other"},
				wantReason: "{other <nil> false} is no variant of [none some]",
			},
			{
				name:       "returns an error at a variant without a payload that the value states one of",
				shape:      either,
				give:       literal.Variant{Name: "none", Payload: 1, HasPayload: true},
				wantPath:   fault.Path{fault.Variant("none")},
				wantReason: "the variant has no payload, and the value states one",
			},
			{
				name:       "returns an error at a variant with a payload that the value states none of",
				shape:      either,
				give:       literal.Variant{Name: "some"},
				wantPath:   fault.Path{fault.Variant("some")},
				wantReason: "the variant has a payload, and the value states none",
			},
			{
				name:       "returns an error at a variant whose payload it does not produce",
				shape:      either,
				give:       literal.Variant{Name: "some", Payload: 300, HasPayload: true},
				wantPath:   fault.Path{fault.Variant("some")},
				wantReason: "300 is outside [0, 255]",
			},
			{
				name:       "returns an error for a literal's value that is none of its values",
				shape:      `{"shape":"literal","values":[{"type":"string","value":"paid"}]}`,
				give:       "lost",
				wantReason: "lost is none of the 1 values",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := engine.Invert(read(t, tt.shape), tt.give)
				isFault(t, err, engine.ErrCannotInvert, tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns an error at a key caused by the fault of the map's keys", func(t *testing.T) {
			t.Parallel()
			_, err := engine.Invert(read(t, bytesToBools), any(map[int]bool{300: true}))
			cause := assert.ErrorAs[*fault.Error](t, assert.ErrorAs[*fault.Error](t, err, "a fault").Err,
				"the fault of the keys")
			assert.Equal(t, cause.Reason, "300 is outside [0, 255]", "why the keys produce no such key")
		})
	})
}

// float returns the float choice of v.
func float(v float64) choice.Choice {
	return choice.Choice{Kind: choice.Float, Float: v}
}
