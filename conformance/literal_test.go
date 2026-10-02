// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestLiteral checks Decode, which turns a typed literal into the native
// value that it states.
func TestLiteral(t *testing.T) {
	t.Parallel()

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want any
		}{
			{name: "returns nil for the null literal", give: `{"type":"null"}`, want: nil},
			{name: "returns a bool for a bool literal", give: `{"type":"bool","value":true}`, want: true},
			{name: "returns an int for an int literal", give: `{"type":"int","value":1}`, want: 1},
			{
				name: "returns an int64 for a decimal string below -(2^53 - 1)",
				give: `{"type":"int","value":"-9007199254740992"}`,
				want: int64(-9007199254740992),
			},
			{
				name: "returns an int64 for a decimal string above 2^53 - 1",
				give: `{"type":"int","value":"9007199254740992"}`,
				want: int64(9007199254740992),
			},
			{
				name: "returns a uint64 for a decimal string above the int64 range",
				give: `{"type":"int","value":"9223372036854775808"}`,
				want: uint64(9223372036854775808),
			},
			{name: "returns a float64 for a float literal", give: `{"type":"float","value":1.5}`, want: 1.5},
			{
				name: "returns positive infinity for the float literal Inf",
				give: `{"type":"float","value":"Inf"}`,
				want: math.Inf(1),
			},
			{
				name: "returns negative infinity for the float literal -Inf",
				give: `{"type":"float","value":"-Inf"}`,
				want: math.Inf(-1),
			},
			{name: "returns a string for a string literal", give: `{"type":"string","value":"abc"}`, want: "abc"},
			{
				name: "returns the bytes of lowercase hexadecimal",
				give: `{"type":"bytes","value":"00ff"}`,
				want: []byte{0, 0xff},
			},
			{
				name: "returns a non-nil empty slice for an empty list",
				give: `{"type":"list","of":"int","value":[]}`,
				want: []int{},
			},
			{
				name: "returns a slice of bool for a list of bool",
				give: `{"type":"list","of":"bool","value":[true]}`,
				want: []bool{true},
			},
			{
				name: "returns a slice of float64 for a list of float",
				give: `{"type":"list","of":"float","value":[1.5]}`,
				want: []float64{1.5},
			},
			{
				name: "returns a slice of string for a list of string",
				give: `{"type":"list","of":"string","value":["a"]}`,
				want: []string{"a"},
			},
			{
				name: "returns a slice of any for a list of items",
				give: `{"type":"list","items":[{"type":"int","value":1},{"type":"string","value":"a"}]}`,
				want: []any{1, "a"},
			},
			{
				name: "returns a non-nil empty map for an empty map of string keys",
				give: `{"type":"map","key":"string","of":"int","value":{}}`,
				want: map[string]int{},
			},
			{
				name: "returns a map of int for a map of string keys and int values",
				give: `{"type":"map","key":"string","of":"int","value":{"a":1}}`,
				want: map[string]int{"a": 1},
			},
			{
				name: "returns a map of bool for a map of string keys and bool values",
				give: `{"type":"map","key":"string","of":"bool","value":{"a":true}}`,
				want: map[string]bool{"a": true},
			},
			{
				name: "returns a map of float64 for a map of string keys and float values",
				give: `{"type":"map","key":"string","of":"float","value":{"a":1.5}}`,
				want: map[string]float64{"a": 1.5},
			},
			{
				name: "returns a map of string for a map of string keys and string values",
				give: `{"type":"map","key":"string","of":"string","value":{"a":"b"}}`,
				want: map[string]string{"a": "b"},
			},
			{
				name: "returns a map of any keys for a map of entries",
				give: `{"type":"map","entries":[[{"type":"int","value":1},{"type":"string","value":"a"}]]}`,
				want: map[any]any{1: "a"},
			},
			{
				name: "returns a map of a nil key for an entry of a null key",
				give: `{"type":"map","entries":[[{"type":"null"},{"type":"int","value":1}]]}`,
				want: map[any]any{nil: 1},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := conformance.Decode(json.RawMessage(tt.give))
				assert.NoError(t, err, "the literal decodes")
				assert.Equal(t, got, tt.want, "the literal decodes to the value of its type")
			})
		}

		t.Run("returns NaN for the float literal NaN", func(t *testing.T) {
			t.Parallel()
			got, err := conformance.Decode(json.RawMessage(`{"type":"float","value":"NaN"}`))
			assert.NoError(t, err, "the literal decodes")
			assert.True(t, math.IsNaN(got.(float64)), "the literal decodes to NaN")
		})

		unknown := []struct {
			name string
			give string
		}{
			{name: "returns ErrUnknownType for an unknown type", give: `{"type":"widget"}`},
			{
				name: "returns ErrUnknownType for a list of an unknown element type",
				give: `{"type":"list","of":"widget","value":[]}`,
			},
			{
				name: "returns ErrUnknownType for a list item of an unknown type",
				give: `{"type":"list","items":[{"type":"widget"}]}`,
			},
			{
				name: "returns ErrUnknownType for a map of int keys stated as an object",
				give: `{"type":"map","key":"int","of":"int","value":{}}`,
			},
			{
				name: "returns ErrUnknownType for a map of string keys and values of an unknown type",
				give: `{"type":"map","key":"string","of":"widget","value":{}}`,
			},
			{
				name: "returns ErrUnknownType for a map key of an unknown type",
				give: `{"type":"map","entries":[[{"type":"widget"},{"type":"int","value":1}]]}`,
			},
			{
				name: "returns ErrUnknownType for a map value of an unknown type",
				give: `{"type":"map","entries":[[{"type":"int","value":1},{"type":"widget"}]]}`,
			},
		}
		for _, tt := range unknown {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := conformance.Decode(json.RawMessage(tt.give))
				assert.ErrorIs(t, err, conformance.ErrUnknownType, "the literal states a type outside the encoding")
			})
		}

		refusals := []struct {
			name string
			give string
			want string
		}{
			{name: "returns an error for text that is no JSON", give: `{`, want: "parse literal"},
			{
				name: "returns an error for an int value that is no number",
				give: `{"type":"int","value":true}`,
				want: "decode scalar",
			},
			{
				name: "returns an error for the decimal string of -(2^53 - 1)",
				give: `{"type":"int","value":"-9007199254740991"}`,
				want: "-9007199254740991 is within 2^53 - 1",
			},
			{
				name: "returns an error for the decimal string of 2^53 - 1",
				give: `{"type":"int","value":"9007199254740991"}`,
				want: "9007199254740991 is within 2^53 - 1",
			},
			{
				name: "returns an error for a decimal string of a plus sign",
				give: `{"type":"int","value":"+9007199254740992"}`,
				want: "is no canonical integer of 64 bits",
			},
			{
				name: "returns an error for a decimal string of a leading zero",
				give: `{"type":"int","value":"09007199254740992"}`,
				want: "is no canonical integer of 64 bits",
			},
			{
				name: "returns an error for a decimal string of 2^64",
				give: `{"type":"int","value":"18446744073709551616"}`,
				want: "is no canonical integer of 64 bits",
			},
			{
				name: "returns an error for an unknown float name",
				give: `{"type":"float","value":"Huge"}`,
				want: `unrecognized float literal "Huge"`,
			},
			{
				name: "returns an error for a float value that is no number",
				give: `{"type":"float","value":true}`,
				want: "decode float",
			},
			{
				name: "returns an error for a bytes value that is no string",
				give: `{"type":"bytes","value":1}`,
				want: "decode bytes",
			},
			{
				name: "returns an error for uppercase hexadecimal",
				give: `{"type":"bytes","value":"00FF"}`,
				want: "is no lowercase hexadecimal",
			},
			{
				name: "returns an error for hexadecimal of an odd length",
				give: `{"type":"bytes","value":"0"}`,
				want: "is no lowercase hexadecimal",
			},
			{
				name: "returns an error for a list value that is no array",
				give: `{"type":"list","of":"int","value":3}`,
				want: "decode list",
			},
			{
				name: "returns an error for a map value that is no object",
				give: `{"type":"map","key":"string","of":"int","value":[]}`,
				want: "decode map",
			},
			{
				name: "returns an error for a map key of a list",
				give: `{"type":"map","entries":[[{"type":"list","of":"int","value":[1]},{"type":"int","value":1}]]}`,
				want: "map key 0 of []int is no key of a Go map",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := conformance.Decode(json.RawMessage(tt.give))
				assert.HasError(t, err, "the literal states no value of its type")
				assert.Contains(t, err.Error(), tt.want, "the error names what the literal misstates")
			})
		}
	})
}
