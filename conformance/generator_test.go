// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// TestGenerator checks the refusal of a generator spec that names no
// generator of the vocabulary, misstates a parameter, or states no domain,
// through the generator of a decoding vector. The definition's vectors run
// every generator of the vocabulary. Written with testing rather than with
// this library, because a verdict is not written with the subject.
func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		at := func(segs ...fault.Segment) fault.Path {
			return inVector(append([]fault.Segment{fault.Field(generatorAt)}, segs...)...)
		}
		const noDomain = "the generator states no domain"
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault for a generator that is no JSON object",
				give:       `3`,
				wantPath:   at(),
				wantReason: "the generator does not parse",
			},
			{
				name:       "returns a fault at gen for a generator of an unknown id",
				give:       unknownGenerator,
				wantPath:   at(fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at gen for self outside a recursive extension",
				give:       `{"gen":"self"}`,
				wantPath:   at(fault.Field(genAt)),
				wantReason: "self is outside a recursive extension",
			},
			{
				name:       "returns a fault at min for an integer of a fractional min",
				give:       `{"gen":"integer","min":0.5,"max":9}`,
				wantPath:   at(fault.Field("min")),
				wantReason: "the value is no int",
			},
			{
				name:       "returns a fault at max for an integer of a fractional max",
				give:       `{"gen":"integer","min":0,"max":9.5}`,
				wantPath:   at(fault.Field("max")),
				wantReason: "the value is no int",
			},
			{
				name:       "returns a fault for integer bounds inside no 64-bit range",
				give:       `{"gen":"integer","min":"-9223372036854775808","max":"18446744073709551615"}`,
				wantPath:   at(),
				wantReason: "the bounds [-9223372036854775808, 18446744073709551615] are inside no 64-bit range",
			},
			{
				name:       "returns a fault for an integer whose min is above its max",
				give:       `{"gen":"integer","min":9,"max":0}`,
				wantPath:   at(),
				wantReason: noDomain,
			},
			{
				name:       "returns a fault at min for a duration of a fractional min",
				give:       `{"gen":"duration","min":0.5,"max":9}`,
				wantPath:   at(fault.Field("min")),
				wantReason: "the value is no int",
			},
			{
				name:       "returns a fault for a duration beyond the int64 range",
				give:       `{"gen":"duration","min":0,"max":"18446744073709551615"}`,
				wantPath:   at(),
				wantReason: "the bounds [0, 18446744073709551615] are no durations",
			},
			{
				name:       "returns a fault at min for a float of a min of an unknown name",
				give:       `{"gen":"float","min":"Huge","max":1}`,
				wantPath:   at(fault.Field("min")),
				wantReason: `"Huge" is none of the names NaN, Inf and -Inf`,
			},
			{
				name:       "returns a fault at max for a float of a max of an unknown name",
				give:       `{"gen":"float","min":0,"max":"Large"}`,
				wantPath:   at(fault.Field("max")),
				wantReason: `"Large" is none of the names NaN, Inf and -Inf`,
			},
			{
				name:       "returns a fault at width for a float of width 16",
				give:       `{"gen":"float","min":0,"max":1,"width":16}`,
				wantPath:   at(fault.Field("width")),
				wantReason: "16 is neither 32 nor 64",
			},
			{
				name:       "returns a fault for a float of width 32 whose min it cannot state",
				give:       `{"gen":"float","min":0.1,"max":1,"width":32}`,
				wantPath:   at(),
				wantReason: "the bounds [0.1, 1] are no floats of width 32",
			},
			{
				name:       "returns a fault for a float of width 32 whose max it cannot state",
				give:       `{"gen":"float","min":0,"max":0.1,"width":32}`,
				wantPath:   at(),
				wantReason: "the bounds [0, 0.1] are no floats of width 32",
			},
			{
				name:       "returns a fault at p for a boolean of three odds",
				give:       `{"gen":"boolean","p":[1,2,3]}`,
				wantPath:   at(fault.Field("p")),
				wantReason: "[1 2 3] is no numerator and denominator",
			},
			{
				name:       "returns a fault for a boolean of odds above one",
				give:       `{"gen":"boolean","p":[3,2]}`,
				wantPath:   at(),
				wantReason: noDomain,
			},
			{
				name:       "returns a fault at the value for just of a literal of an unknown type",
				give:       `{"gen":"just","value":` + widget + `}`,
				wantPath:   at(fault.Field(valueAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at the values for sampled-from of values that are no list",
				give:       `{"gen":"sampled-from","values":3}`,
				wantPath:   at(fault.Field("values")),
				wantReason: "the values are no list",
			},
			{
				name:       "returns a fault at the value for permutation of a literal of an unknown type",
				give:       `{"gen":"permutation","values":[` + widget + `]}`,
				wantPath:   at(fault.Field("values"), fault.Index(0), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault for sampled-from of no value",
				give:       `{"gen":"sampled-from","values":[]}`,
				wantPath:   at(),
				wantReason: noDomain,
			},
			{
				name:       "returns a fault at of for one-of of generators that are no list",
				give:       `{"gen":"one-of","of":3}`,
				wantPath:   at(fault.Field(ofAt)),
				wantReason: "the generators are no list",
			},
			{
				name:       "returns a fault at the generator for one-of of a generator of an unknown id",
				give:       `{"gen":"one-of","of":[` + digitGenerator + `,` + unknownGenerator + `]}`,
				wantPath:   at(fault.Field(ofAt), fault.Index(1), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at of for optional of a generator of an unknown id",
				give:       `{"gen":"optional","of":` + unknownGenerator + `}`,
				wantPath:   at(fault.Field(ofAt), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at keep for a filter by a predicate of an unknown kind",
				give:       `{"gen":"filter","of":` + digitGenerator + `,"keep":{"kind":"most"}}`,
				wantPath:   at(fault.Field("keep"), fault.Field(kindAt)),
				wantReason: `"most" names no predicate`,
			},
			{
				name:       "returns a fault for a list whose min_size is above its max_size",
				give:       `{"gen":"list","of":` + digitGenerator + `,"min_size":3,"max_size":1}`,
				wantPath:   at(),
				wantReason: "the sizes [3, 1] admit no size",
			},
			{
				name:       "returns a fault at keys for a dict of keys of an unknown id",
				give:       `{"gen":"dict","keys":` + unknownGenerator + `,"values":` + digitGenerator + `}`,
				wantPath:   at(fault.Field("keys"), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at the values for a dict of values of an unknown id",
				give:       `{"gen":"dict","keys":` + digitGenerator + `,"values":` + unknownGenerator + `}`,
				wantPath:   at(fault.Field("values"), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault for a dict of a negative min_size",
				give:       `{"gen":"dict","keys":` + digitGenerator + `,"values":` + digitGenerator + `,"min_size":-1}`,
				wantPath:   at(),
				wantReason: "the minimum size -1 is negative",
			},
			{
				name:       "returns a fault at the pattern for a pattern outside the portable subset",
				give:       `{"gen":"string-matching","pattern":"("}`,
				wantPath:   at(fault.Field("pattern")),
				wantReason: `"(" at 1: a group is not closed`,
			},
			{
				name:       "returns a fault for bytes whose min_size is above their max_size",
				give:       `{"gen":"bytes","min_size":2,"max_size":1}`,
				wantPath:   at(),
				wantReason: "the sizes [2, 1] admit no size",
			},
			{
				name:       "returns a fault at the base for a recursive base of an unknown id",
				give:       `{"gen":"recursive","base":` + unknownGenerator + `,"extend":{"gen":"self"}}`,
				wantPath:   at(fault.Field("base"), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at the extension for a recursive extension of an unknown id",
				give:       `{"gen":"recursive","base":` + digitGenerator + `,"extend":` + unknownGenerator + `}`,
				wantPath:   at(fault.Field("extend"), fault.Field(genAt)),
				wantReason: noGenerator,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := check(t, conformance.Decoding, decoded(tt.give, `[]`, `[]`, null))
				expectFault(t, err, tt.wantPath, tt.wantReason)
			})
		}
	})
}
