// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// unknownGenerator is a generator spec of an id outside the vocabulary.
const unknownGenerator = `{"gen":"widget"}`

// TestGenerator checks the refusal of a generator spec that names no
// generator of the vocabulary, misstates a parameter, or states no domain.
// The definition's vectors run every generator of the vocabulary.
func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns an error for a generator that is no JSON object",
				give: `3`,
				want: "parse generator",
			},
			{
				name: "returns an error for a generator of an unknown id",
				give: unknownGenerator,
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for self outside a recursive extension",
				give: `{"gen":"self"}`,
				want: "self is outside a recursive extension",
			},
			{
				name: "returns an error for an integer of a fractional min",
				give: `{"gen":"integer","min":0.5,"max":9}`,
				want: "integer min",
			},
			{
				name: "returns an error for an integer of a fractional max",
				give: `{"gen":"integer","min":0,"max":9.5}`,
				want: "integer max",
			},
			{
				name: "returns an error for integer bounds inside no 64-bit range",
				give: `{"gen":"integer","min":"-9223372036854775808","max":"18446744073709551615"}`,
				want: "lies inside no 64-bit range",
			},
			{
				name: "returns an error for an integer whose min is above its max",
				give: `{"gen":"integer","min":9,"max":0}`,
				want: "states no domain",
			},
			{
				name: "returns an error for a duration of a fractional min",
				give: `{"gen":"duration","min":0.5,"max":9}`,
				want: "duration min",
			},
			{
				name: "returns an error for a duration beyond the int64 range",
				give: `{"gen":"duration","min":0,"max":"18446744073709551615"}`,
				want: "are no durations",
			},
			{
				name: "returns an error for a float of a min of an unknown name",
				give: `{"gen":"float","min":"Huge","max":1}`,
				want: `unrecognized float literal "Huge"`,
			},
			{
				name: "returns an error for a float of a max of an unknown name",
				give: `{"gen":"float","min":0,"max":"Large"}`,
				want: `unrecognized float literal "Large"`,
			},
			{
				name: "returns an error for a float of width 16",
				give: `{"gen":"float","min":0,"max":1,"width":16}`,
				want: "a float of width 16",
			},
			{
				name: "returns an error for a float of width 32 whose min it cannot state",
				give: `{"gen":"float","min":0.1,"max":1,"width":32}`,
				want: "are no floats of width 32",
			},
			{
				name: "returns an error for a float of width 32 whose max it cannot state",
				give: `{"gen":"float","min":0,"max":0.1,"width":32}`,
				want: "are no floats of width 32",
			},
			{
				name: "returns an error for a boolean of three odds",
				give: `{"gen":"boolean","p":[1,2,3]}`,
				want: "p is [1 2 3], not [numerator, denominator]",
			},
			{
				name: "returns an error for a boolean of odds above one",
				give: `{"gen":"boolean","p":[3,2]}`,
				want: "states no domain",
			},
			{
				name: "returns an error for just of a literal of an unknown type",
				give: `{"gen":"just","value":{"type":"widget"}}`,
				want: conformance.ErrUnknownType.Error(),
			},
			{
				name: "returns an error for sampled-from of values that are no list",
				give: `{"gen":"sampled-from","values":3}`,
				want: "sampled-from values",
			},
			{
				name: "returns an error for permutation of a literal of an unknown type",
				give: `{"gen":"permutation","values":[{"type":"widget"}]}`,
				want: conformance.ErrUnknownType.Error(),
			},
			{
				name: "returns an error for sampled-from of no value",
				give: `{"gen":"sampled-from","values":[]}`,
				want: "states no domain",
			},
			{
				name: "returns an error for one-of of generators that are no list",
				give: `{"gen":"one-of","of":3}`,
				want: "one-of of",
			},
			{
				name: "returns an error for one-of of a generator of an unknown id",
				give: `{"gen":"one-of","of":[` + unknownGenerator + `]}`,
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for optional of a generator of an unknown id",
				give: `{"gen":"optional","of":` + unknownGenerator + `}`,
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for a filter by a predicate of an unknown kind",
				give: `{"gen":"filter","of":` + digitGenerator + `,"keep":{"kind":"most"}}`,
				want: `"most" names no predicate`,
			},
			{
				name: "returns an error for a list whose min_size is above its max_size",
				give: `{"gen":"list","of":` + digitGenerator + `,"min_size":3,"max_size":1}`,
				want: "sizes [3, 1]",
			},
			{
				name: "returns an error for a dict of keys of an unknown id",
				give: `{"gen":"dict","keys":` + unknownGenerator + `,"values":` + digitGenerator + `}`,
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for a dict of values of an unknown id",
				give: `{"gen":"dict","keys":` + digitGenerator + `,"values":` + unknownGenerator + `}`,
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for a dict of a negative min_size",
				give: `{"gen":"dict","keys":` + digitGenerator + `,"values":` + digitGenerator + `,"min_size":-1}`,
				want: "minimum size -1",
			},
			{
				name: "returns an error for a pattern outside the portable subset",
				give: `{"gen":"string-matching","pattern":"("}`,
				want: "pattern: outside the portable subset",
			},
			{
				name: "returns an error for bytes whose min_size is above their max_size",
				give: `{"gen":"bytes","min_size":2,"max_size":1}`,
				want: "sizes [2, 1]",
			},
			{
				name: "returns an error for a recursive base of an unknown id",
				give: `{"gen":"recursive","base":` + unknownGenerator + `,"extend":{"gen":"self"}}`,
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for a recursive extension of an unknown id",
				give: `{"gen":"recursive","base":` + digitGenerator + `,"extend":` + unknownGenerator + `}`,
				want: `"widget" names no generator`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Decoding, decoded(tt.give, `[]`, `[]`, null)), tt.want)
			})
		}
	})
}
