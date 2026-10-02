// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// TestCanonical checks the equality that Check compares a decoded value
// with its literal under: one type for every integer type and for every
// float type, floats compared by their bits, and maps compared by their
// entries in any order.
func TestCanonical(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			generator string
			choices   string
			value     string
			want      string
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
				name:      "returns an error for an int that the literal states as a float",
				generator: `{"gen":"just","value":{"type":"int","value":1}}`,
				choices:   `[]`,
				value:     `{"type":"float","value":1.0}`,
				want:      "the value is int:1",
			},
			{
				name:      "returns an error for a negative zero that the literal states as zero",
				generator: `{"gen":"just","value":{"type":"float","value":-0.0}}`,
				choices:   `[]`,
				value:     `{"type":"float","value":0.0}`,
				want:      "the value is float:-0",
			},
			{
				name:      "returns an error for a float that the literal states with another second digit",
				generator: `{"gen":"just","value":{"type":"float","value":1.25}}`,
				choices:   `[]`,
				value:     `{"type":"float","value":1.3}`,
				want:      "the value is float:1.25",
			},
			{
				name:      "returns an error for a list that the literal states with another element",
				generator: `{"gen":"just","value":{"type":"list","of":"int","value":[1,2]}}`,
				choices:   `[]`,
				value:     `{"type":"list","of":"int","value":[1,3]}`,
				want:      "the value is list:[int:1,int:2]",
			},
			{
				name:      "returns an error for bytes that the literal states as a list of ints",
				generator: `{"gen":"bytes","max_size":2}`,
				choices:   `[{"sequence":[1,2]}]`,
				value:     `{"type":"list","of":"int","value":[1,2]}`,
				want:      "the value is bytes:0102",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				vector := decoded(tt.generator, tt.choices, tt.choices, tt.value)
				expectCheck(t, check(t, conformance.Decoding, vector), tt.want)
			})
		}
	})
}
