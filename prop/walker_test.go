// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	bignum "math/big"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/prop"
)

// keyless is a map whose key shape decodes to bytes, which no Go map
// accepts as keys.
const keyless = `{"shape":"map","key":{"shape":"bytes"},"of":{"shape":"bool"}}`

// keylessReason is the reason of the fault at the key of a map that no Go
// map accepts.
const keylessReason = "the shape decodes to values that no Go map accepts as keys"

// TestWalker checks the Go value that OfShape decodes each structural shape
// to, and each map whose key shape decodes to values that no Go map accepts
// as keys, with the path through the shape file to the key.
func TestWalker(t *testing.T) {
	t.Parallel()

	t.Run("OfShape", func(t *testing.T) {
		t.Parallel()

		decodes := []struct {
			name  string
			shape string
			give  []choice.Choice
			want  any
		}{
			{
				name:  "decodes a signed int of 8 bits to an int8",
				shape: `{"shape":"int","width":8,"signed":true}`,
				give:  []choice.Choice{signed(-5)},
				want:  int8(-5),
			},
			{
				name:  "decodes an unsigned int of 16 bits to a uint16",
				shape: `{"shape":"int","width":16,"signed":false}`,
				give:  []choice.Choice{integer(7)},
				want:  uint16(7),
			},
			{
				name:  "decodes an int of 128 bits to a *big.Int",
				shape: `{"shape":"int","width":128,"signed":true}`,
				give:  []choice.Choice{signed(0), integer(9)},
				want:  bignum.NewInt(9),
			},
			{
				name:  "decodes a float of 32 bits to a float32",
				shape: `{"shape":"float","width":32}`,
				give:  []choice.Choice{floating(1.5)},
				want:  float32(1.5),
			},
			{
				name:  "decodes a float of 64 bits to a float64",
				shape: `{"shape":"float","width":64}`,
				give:  []choice.Choice{floating(1.5)},
				want:  1.5,
			},
			{
				name:  "decodes a bool to a bool",
				shape: `{"shape":"bool"}`,
				give:  []choice.Choice{integer(1)},
				want:  true,
			},
			{
				name:  "decodes a string to a string",
				shape: `{"shape":"string"}`,
				give:  []choice.Choice{sequence(10)},
				want:  "a",
			},
			{
				name:  "decodes bytes to a []byte",
				shape: `{"shape":"bytes"}`,
				give:  []choice.Choice{sequence(1, 2)},
				want:  []byte{1, 2},
			},
			{
				name:  "decodes a list to a []any",
				shape: `{"shape":"list","of":{"shape":"bool"}}`,
				give:  []choice.Choice{integer(1), integer(1), integer(0)},
				want:  []any{true},
			},
			{
				name:  "decodes a list of no element to a nil []any",
				shape: `{"shape":"list","of":{"shape":"bool"}}`,
				give:  []choice.Choice{integer(0)},
				want:  []any(nil),
			},
			{
				name:  "decodes a fixed-list to a []any",
				shape: `{"shape":"fixed-list","of":{"shape":"bool"},"size":2}`,
				give:  []choice.Choice{integer(1), integer(1), integer(1), integer(0), integer(0)},
				want:  []any{true, false},
			},
			{
				name:  "decodes a set to a []any",
				shape: `{"shape":"set","of":{"shape":"string"}}`,
				give:  []choice.Choice{integer(1), sequence(10), integer(0)},
				want:  []any{"a"},
			},
			{
				name:  "decodes a map to a map[any]any",
				shape: `{"shape":"map","key":{"shape":"string"},"of":{"shape":"bool"}}`,
				give:  []choice.Choice{integer(1), sequence(10), integer(1), integer(0)},
				want:  map[any]any{"a": true},
			},
			{
				name:  "decodes an absent optional to nil",
				shape: `{"shape":"optional","of":{"shape":"bool"}}`,
				give:  []choice.Choice{integer(0)},
				want:  nil,
			},
			{
				name:  "decodes a present optional to its value",
				shape: `{"shape":"optional","of":{"shape":"bool"}}`,
				give:  []choice.Choice{integer(1), integer(1)},
				want:  true,
			},
			{
				name:  "decodes a record to a map[string]any of its fields",
				shape: `{"shape":"record","fields":[["on",{"shape":"bool"}],["n",{"shape":"int","width":8,"signed":false}]]}`,
				give:  []choice.Choice{integer(1), integer(7)},
				want:  map[string]any{"on": true, "n": uint8(7)},
			},
			{
				name:  "decodes a literal of a record to a map[string]any",
				shape: `{"shape":"literal","values":[{"type":"record","fields":[["a",{"type":"int","value":1}]]}]}`,
				give:  []choice.Choice{integer(0)},
				want:  map[string]any{"a": 1},
			},
			{
				name: "decodes a literal of a variant to a Variant of its payload",
				shape: `{"shape":"literal","values":[{"type":"null"},` +
					`{"type":"variant","name":"v","payload":{"type":"list","items":[{"type":"int","value":2}]}}]}`,
				give: []choice.Choice{integer(1)},
				want: prop.Variant{Name: "v", Payload: []any{2}, HasPayload: true},
			},
			{
				name:  "decodes a literal of null to nil",
				shape: `{"shape":"literal","values":[{"type":"null"},{"type":"int","value":1}]}`,
				give:  []choice.Choice{integer(0)},
				want:  nil,
			},
			{
				name: "decodes a literal of a map to a map of each value's Go value",
				shape: `{"shape":"literal","values":[{"type":"map","entries":[` +
					`[{"type":"int","value":1},{"type":"record","fields":[["a",{"type":"bool","value":true}]]}]]}]}`,
				give: []choice.Choice{integer(0)},
				want: map[any]any{1: map[string]any{"a": true}},
			},
			{
				name:  "decodes a definition that refers to itself",
				shape: `{"shape":"ref","name":"t","definitions":{"t":{"shape":"list","of":{"shape":"ref","name":"t"}}}}`,
				give:  []choice.Choice{integer(1), integer(0), integer(0)},
				want:  []any{[]any(nil)},
			},
			{
				name: "decodes two refs to one definition by the one converter",
				shape: `{"shape":"record","fields":[["a",{"shape":"ref","name":"d"}],["b",{"shape":"ref","name":"d"}]],` +
					`"definitions":{"d":{"shape":"bool"}}}`,
				give: []choice.Choice{integer(1), integer(0)},
				want: map[string]any{"a": true, "b": false},
			},
		}
		for _, tt := range decodes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				g, err := prop.OfShape(tt.shape)
				assert.NoError(t, err, "the shape reads")
				assert.Equal(t, decodedBy(g, tt.give...), tt.want, "the Go value of the shape")
			})
		}

		keys := []struct {
			name string
			give string
		}{
			{name: "returns a fault at the key of a map keyed by bytes", give: `{"shape":"bytes"}`},
			{
				name: "returns a fault at the key of a map keyed by lists",
				give: `{"shape":"list","of":{"shape":"bool"}}`,
			},
			{
				name: "returns a fault at the key of a map keyed by records",
				give: `{"shape":"record","fields":[["a",{"shape":"bool"}]]}`,
			},
			{
				name: "returns a fault at the key of a map keyed by optional sets",
				give: `{"shape":"optional","of":{"shape":"set","of":{"shape":"bool"}}}`,
			},
			{
				name: "returns a fault at the key of a map keyed by a variant of a fixed-list",
				give: `{"shape":"enum","variants":[["none",null],["many",{"shape":"fixed-list","size":1,"of":{"shape":"bool"}}]]}`,
			},
			{
				name: "returns a fault at the key of a map keyed by a literal of a list",
				give: `{"shape":"literal","values":[{"type":"null"},{"type":"list","items":[{"type":"int","value":1}]}]}`,
			},
			{
				name: "returns a fault at the key of a map keyed by a definition of maps",
				give: `{"shape":"ref","name":"m"}`,
			},
		}
		for _, tt := range keys {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := prop.OfShape(`{"shape":"map","key":` + tt.give + `,"of":{"shape":"bool"},` +
					`"definitions":{"m":{"shape":"map","key":{"shape":"string"},"of":{"shape":"bool"}}}}`)
				expectFault(
					t,
					err,
					fault.Error{Op: ofShapeOp, Path: fault.Path{fault.Field("key")}, Reason: keylessReason},
				)
			})
		}

		nested := []struct {
			name     string
			give     string
			wantPath fault.Path
		}{
			{
				name:     "returns a fault on the path of a key inside a list",
				give:     `{"shape":"list","of":` + keyless + `}`,
				wantPath: fault.Path{fault.Field("of"), fault.Field("key")},
			},
			{
				name:     "returns a fault on the path of a key inside a set",
				give:     `{"shape":"set","of":` + keyless + `}`,
				wantPath: fault.Path{fault.Field("of"), fault.Field("key")},
			},
			{
				name:     "returns a fault on the path of a key inside a map's key",
				give:     `{"shape":"map","key":` + keyless + `,"of":{"shape":"bool"}}`,
				wantPath: fault.Path{fault.Field("key"), fault.Field("key")},
			},
			{
				name:     "returns a fault on the path of a key inside a map's value",
				give:     `{"shape":"map","key":{"shape":"string"},"of":` + keyless + `}`,
				wantPath: fault.Path{fault.Field("of"), fault.Field("key")},
			},
			{
				name:     "returns a fault on the path of a key inside an optional",
				give:     `{"shape":"optional","of":` + keyless + `}`,
				wantPath: fault.Path{fault.Field("of"), fault.Field("key")},
			},
			{
				name:     "returns a fault on the path of a key inside a record's field",
				give:     `{"shape":"record","fields":[["a",{"shape":"bool"}],["m",` + keyless + `]]}`,
				wantPath: fault.Path{fault.Field("fields"), fault.Index(1), fault.Index(1), fault.Field("key")},
			},
			{
				name:     "returns a fault on the path of a key inside a variant's payload",
				give:     `{"shape":"enum","variants":[["none",null],["v",` + keyless + `]]}`,
				wantPath: fault.Path{fault.Field("variants"), fault.Index(1), fault.Index(1), fault.Field("key")},
			},
			{
				name:     "returns a fault on the path of a key inside a definition",
				give:     `{"shape":"ref","name":"d","definitions":{"d":` + keyless + `}}`,
				wantPath: fault.Path{fault.Field("definitions"), fault.Key("d"), fault.Field("key")},
			},
		}
		for _, tt := range nested {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := prop.OfShape(tt.give)
				expectFault(t, err, fault.Error{Op: ofShapeOp, Path: tt.wantPath, Reason: keylessReason})
			})
		}

		t.Run("reads a map keyed by values that a Go map accepts as keys", func(t *testing.T) {
			t.Parallel()
			text := `{"shape":"record","fields":[` +
				`["a",{"shape":"map","key":{"shape":"optional","of":{"shape":"string"}},"of":{"shape":"bool"}}],` +
				`["b",{"shape":"map","key":{"shape":"enum","variants":[["none",null],["some",{"shape":"bool"}]]},` +
				`"of":{"shape":"bool"}}],` +
				`["c",{"shape":"map","key":{"shape":"literal","values":[{"type":"null"},{"type":"int","value":1}]},` +
				`"of":{"shape":"bool"}}],` +
				`["d",{"shape":"map","key":{"shape":"ref","name":"k"},"of":{"shape":"bool"}}]],` +
				`"definitions":{"k":{"shape":"optional","of":{"shape":"ref","name":"k"}}}}`
			_, err := prop.OfShape(text)
			assert.NoError(t, err, "a Go map accepts every key")
		})
	})
}
