// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// rejectingGenerator is a filter that rejects its one value at every
// attempt, and so rejects the case.
const rejectingGenerator = `{"gen":"filter","of":{"gen":"just","value":{"type":"int","value":1}},` +
	`"keep":{"kind":"never"}}`

// TestDecoded checks the comparison of a decoded case with the one that a
// vector states, through decoding vectors: the rejection, and the value
// under the equality of canonical text, which makes every integer type one
// type and every float type another, compares a float by its bits, and a
// map by its entries in any order. Written with testing rather than with
// this library, because a verdict is not written with the subject.
func TestDecoded(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		atRejected := inVector(fault.Field("rejected"))
		atValue := inVector(fault.Field(valueAt))
		tests := []struct {
			name       string
			generator  string
			choices    string
			value      string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:      "returns nil for an int64 that the literal states as an int",
				generator: digitGenerator,
				choices:   `[7]`,
				value:     `{"type":"int","value":7}`,
			},
			{
				name:      "returns nil for a duration that the literal states as an int",
				generator: `{"gen":"duration","min":0,"max":9}`,
				choices:   `[7]`,
				value:     `{"type":"int","value":7}`,
			},
			{
				name:      "returns nil for a float32 that the literal states as a float",
				generator: `{"gen":"float","min":0,"max":2,"width":32}`,
				choices:   `[{"float":1.5}]`,
				value:     `{"type":"float","value":1.5}`,
			},
			{
				name:      "returns nil for a NaN that the literal states as NaN",
				generator: `{"gen":"float","min":0,"max":1,"allow_nan":true}`,
				choices:   `[{"float":"NaN"}]`,
				value:     `{"type":"float","value":"NaN"}`,
			},
			{
				name:      "returns nil for a map of string keys that the literal states as entries in another order",
				generator: `{"gen":"just","value":{"type":"map","key":"string","of":"int","value":{"a":1,"b":2}}}`,
				choices:   `[]`,
				value: `{"type":"map","entries":[` +
					`[{"type":"string","value":"b"},{"type":"int","value":2}],` +
					`[{"type":"string","value":"a"},{"type":"int","value":1}]]}`,
			},
			{
				name:      "returns nil for an absent optional in a list that the literal states as null",
				generator: `{"gen":"list","of":{"gen":"optional","of":{"gen":"integer","min":1,"max":9}},"max_size":2}`,
				choices:   `[1,0,1,1,5,0]`,
				value:     `{"type":"list","items":[{"type":"null"},{"type":"int","value":5}]}`,
			},
			{
				name:       "returns a fault at the value for an int that the literal states as a float",
				generator:  `{"gen":"just","value":{"type":"int","value":1}}`,
				choices:    `[]`,
				value:      `{"type":"float","value":1.0}`,
				wantPath:   atValue,
				wantReason: `the value is int:1, want {"type":"float","value":1.0}`,
			},
			{
				name:       "returns a fault at the value for a negative zero that the literal states as zero",
				generator:  `{"gen":"just","value":{"type":"float","value":-0.0}}`,
				choices:    `[]`,
				value:      `{"type":"float","value":0.0}`,
				wantPath:   atValue,
				wantReason: `the value is float:-0, want {"type":"float","value":0.0}`,
			},
			{
				name:       "returns a fault at the value for a float that the literal states with another second digit",
				generator:  `{"gen":"just","value":{"type":"float","value":1.25}}`,
				choices:    `[]`,
				value:      `{"type":"float","value":1.3}`,
				wantPath:   atValue,
				wantReason: `the value is float:1.25, want {"type":"float","value":1.3}`,
			},
			{
				name:       "returns a fault at the value for a list that the literal states with another element",
				generator:  `{"gen":"just","value":{"type":"list","of":"int","value":[1,2]}}`,
				choices:    `[]`,
				value:      `{"type":"list","of":"int","value":[1,3]}`,
				wantPath:   atValue,
				wantReason: `the value is list:[int:1,int:2], want {"type":"list","of":"int","value":[1,3]}`,
			},
			{
				name:       "returns a fault at the value for bytes that the literal states as a list of ints",
				generator:  `{"gen":"bytes","max_size":2}`,
				choices:    `[{"sequence":[1,2]}]`,
				value:      `{"type":"list","of":"int","value":[1,2]}`,
				wantPath:   atValue,
				wantReason: `the value is bytes:0102, want {"type":"list","of":"int","value":[1,2]}`,
			},
			{
				name:       "returns a fault at the value for a value of another type than the literal states",
				generator:  digitGenerator,
				choices:    `[7]`,
				value:      `{"type":"string","value":"7"}`,
				wantPath:   atValue,
				wantReason: `the value is int:7, want {"type":"string","value":"7"}`,
			},
			{
				name:       "returns a fault at rejected for a decoded case that the vector states as rejected",
				generator:  digitGenerator,
				choices:    `[7]`,
				value:      null,
				wantPath:   atRejected,
				wantReason: "the rejection is false, want true",
			},
			{
				name:       "returns a fault at rejected for a rejected case that the vector states as decoded",
				generator:  rejectingGenerator,
				choices:    `[]`,
				value:      `{"type":"int","value":1}`,
				wantPath:   atRejected,
				wantReason: "the rejection is true, want false",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				vector := decoded(tt.generator, tt.choices, tt.choices, tt.value)
				expectFault(t, check(t, conformance.Decoding, vector), tt.wantPath, tt.wantReason)
			})
		}
	})
}
