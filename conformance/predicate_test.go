// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// pairLiteral is the list of the ints 1 and 2, which the predicates are
// tested on with oneLiteral.
const pairLiteral = `{"type":"list","of":"int","value":[1,2]}`

// TestPredicate checks each predicate of the vocabulary through a filter of
// one stated value: the case keeps the value where the predicate reports
// true, and is rejected where it reports false. Written with testing rather
// than with this library, because a verdict is not written with the
// subject.
func TestPredicate(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			value     string
			predicate string
			holds     bool
		}{
			{
				name:      "keeps any value under always",
				value:     oneLiteral,
				predicate: `{"kind":"always"}`,
				holds:     true,
			},
			{
				name:      "rejects any value under never",
				value:     oneLiteral,
				predicate: `{"kind":"never"}`,
			},
			{
				name:      "rejects any value under a negated always",
				value:     oneLiteral,
				predicate: `{"kind":"always","not":true}`,
			},
			{
				name:      "keeps a value equal to the literal of equals",
				value:     oneLiteral,
				predicate: `{"kind":"equals","value":{"type":"int","value":1}}`,
				holds:     true,
			},
			{
				name:      "rejects an int under equals of a float of the same number",
				value:     oneLiteral,
				predicate: `{"kind":"equals","value":{"type":"float","value":1.0}}`,
			},
			{
				name:      "keeps an int equal to the n of at-least",
				value:     `{"type":"int","value":5}`,
				predicate: `{"kind":"at-least","n":5}`,
				holds:     true,
			},
			{
				name:      "rejects an int below the n of at-least",
				value:     `{"type":"int","value":4}`,
				predicate: `{"kind":"at-least","n":5}`,
			},
			{
				name:      "keeps a float equal to a fractional n of at-least",
				value:     `{"type":"float","value":2.5}`,
				predicate: `{"kind":"at-least","n":2.5}`,
				holds:     true,
			},
			{
				name:      "keeps the largest uint64 under at-least of its decimal string",
				value:     `{"type":"int","value":"18446744073709551615"}`,
				predicate: `{"kind":"at-least","n":"18446744073709551615"}`,
				holds:     true,
			},
			{
				name:      "rejects a NaN under at-least",
				value:     `{"type":"float","value":"NaN"}`,
				predicate: `{"kind":"at-least","n":0}`,
			},
			{
				name:      "rejects a bool under at-least",
				value:     `{"type":"bool","value":true}`,
				predicate: `{"kind":"at-least","n":0}`,
			},
			{
				name:      "keeps an int that the n of divisible-by divides",
				value:     `{"type":"int","value":4}`,
				predicate: `{"kind":"divisible-by","n":2}`,
				holds:     true,
			},
			{
				name:      "rejects an int that the n of divisible-by does not divide",
				value:     `{"type":"int","value":3}`,
				predicate: `{"kind":"divisible-by","n":2}`,
			},
			{
				name:      "rejects any int under divisible-by zero",
				value:     `{"type":"int","value":0}`,
				predicate: `{"kind":"divisible-by","n":0}`,
			},
			{
				name:      "rejects a float of an integer value under divisible-by",
				value:     `{"type":"float","value":4.0}`,
				predicate: `{"kind":"divisible-by","n":2}`,
			},
			{
				name:      "keeps a list whose sum is above the n of sum-above",
				value:     pairLiteral,
				predicate: `{"kind":"sum-above","n":2}`,
				holds:     true,
			},
			{
				name:      "rejects a list whose sum equals the n of sum-above",
				value:     pairLiteral,
				predicate: `{"kind":"sum-above","n":3}`,
			},
			{
				name:      "sums each true of a list as one",
				value:     `{"type":"list","of":"bool","value":[true,true,true]}`,
				predicate: `{"kind":"sum-above","n":2}`,
				holds:     true,
			},
			{
				name:      "sums each false of a list as zero",
				value:     `{"type":"list","of":"bool","value":[false,false]}`,
				predicate: `{"kind":"sum-above","n":0}`,
			},
			{
				name:      "sums a float term after an integer term as a float",
				value:     `{"type":"list","items":[{"type":"int","value":1},{"type":"float","value":1.5}]}`,
				predicate: `{"kind":"sum-above","n":2}`,
				holds:     true,
			},
			{
				name:      "sums an integer term after a float term as a float",
				value:     `{"type":"list","items":[{"type":"float","value":1.5},{"type":"int","value":1}]}`,
				predicate: `{"kind":"sum-above","n":2}`,
				holds:     true,
			},
			{
				name: "sums integers beyond 2^53 exactly",
				value: `{"type":"list","items":[{"type":"int","value":"9007199254740993"},` +
					`{"type":"int","value":0}]}`,
				predicate: `{"kind":"sum-above","n":"9007199254740992"}`,
				holds:     true,
			},
			{
				name:      "skips a string term of a list",
				value:     `{"type":"list","items":[{"type":"string","value":"a"},{"type":"int","value":3}]}`,
				predicate: `{"kind":"sum-above","n":2}`,
				holds:     true,
			},
			{
				name:      "rejects a list whose float sum equals the n of sum-above",
				value:     `{"type":"list","items":[{"type":"float","value":1.5},{"type":"int","value":1}]}`,
				predicate: `{"kind":"sum-above","n":2.5}`,
			},
			{
				name:      "rejects a list whose sum is NaN under sum-above",
				value:     `{"type":"list","items":[{"type":"float","value":"NaN"}]}`,
				predicate: `{"kind":"sum-above","n":"-Inf"}`,
			},
			{
				name:      "rejects a byte string under sum-above",
				value:     `{"type":"bytes","value":"0102"}`,
				predicate: `{"kind":"sum-above","n":0}`,
			},
			{
				name:      "rejects a string under sum-above",
				value:     `{"type":"string","value":"12"}`,
				predicate: `{"kind":"sum-above","n":0}`,
			},
			{
				name:      "keeps a list of n elements under length-at-least",
				value:     pairLiteral,
				predicate: `{"kind":"length-at-least","n":2}`,
				holds:     true,
			},
			{
				name:      "rejects a list of fewer than n elements under length-at-least",
				value:     pairLiteral,
				predicate: `{"kind":"length-at-least","n":3}`,
			},
			{
				name:      "keeps an empty list under length-at-least zero",
				value:     `{"type":"list","of":"int","value":[]}`,
				predicate: `{"kind":"length-at-least","n":0}`,
				holds:     true,
			},
			{
				name:      "rejects a string of six bytes in five characters under length-at-least six",
				value:     `{"type":"string","value":"héllo"}`,
				predicate: `{"kind":"length-at-least","n":6}`,
			},
			{
				name:      "keeps a map of n entries under length-at-least",
				value:     `{"type":"map","key":"string","of":"int","value":{"a":1}}`,
				predicate: `{"kind":"length-at-least","n":1}`,
				holds:     true,
			},
			{
				name:      "rejects an int under length-at-least zero",
				value:     oneLiteral,
				predicate: `{"kind":"length-at-least","n":0}`,
			},
			{
				name:      "keeps a list that contains the literal of contains",
				value:     pairLiteral,
				predicate: `{"kind":"contains","value":{"type":"int","value":2}}`,
				holds:     true,
			},
			{
				name:      "rejects a list without the literal of contains",
				value:     pairLiteral,
				predicate: `{"kind":"contains","value":{"type":"int","value":3}}`,
			},
			{
				name:      "keeps a string that contains the string of contains",
				value:     `{"type":"string","value":"hello"}`,
				predicate: `{"kind":"contains","value":{"type":"string","value":"ell"}}`,
				holds:     true,
			},
			{
				name:      "rejects a string under contains of an int",
				value:     `{"type":"string","value":"1"}`,
				predicate: `{"kind":"contains","value":{"type":"int","value":1}}`,
			},
			{
				name:      "keeps a byte string that contains the bytes of contains",
				value:     `{"type":"bytes","value":"010203"}`,
				predicate: `{"kind":"contains","value":{"type":"bytes","value":"0203"}}`,
				holds:     true,
			},
			{
				name:      "rejects a byte string under contains of a string",
				value:     `{"type":"bytes","value":"6869"}`,
				predicate: `{"kind":"contains","value":{"type":"string","value":"hi"}}`,
			},
			{
				name:      "rejects an int under contains of the same int",
				value:     oneLiteral,
				predicate: `{"kind":"contains","value":{"type":"int","value":1}}`,
			},
			{
				name:      "keeps a list of a number below the one before it under not-sorted",
				value:     `{"type":"list","of":"int","value":[2,1]}`,
				predicate: `{"kind":"not-sorted"}`,
				holds:     true,
			},
			{
				name:      "rejects a list of equal numbers under not-sorted",
				value:     `{"type":"list","items":[{"type":"int","value":1},{"type":"float","value":1.0}]}`,
				predicate: `{"kind":"not-sorted"}`,
			},
			{
				name:      "keeps a list of a string below the one before it under not-sorted",
				value:     `{"type":"list","of":"string","value":["b","a"]}`,
				predicate: `{"kind":"not-sorted"}`,
				holds:     true,
			},
			{
				name:      "rejects a list of equal strings under not-sorted",
				value:     `{"type":"list","of":"string","value":["a","a"]}`,
				predicate: `{"kind":"not-sorted"}`,
			},
			{
				name:      "keeps a list of false after true under not-sorted",
				value:     `{"type":"list","of":"bool","value":[true,false]}`,
				predicate: `{"kind":"not-sorted"}`,
				holds:     true,
			},
			{
				name:      "rejects a list of true after false under not-sorted",
				value:     `{"type":"list","of":"bool","value":[false,true]}`,
				predicate: `{"kind":"not-sorted"}`,
			},
			{
				name:      "rejects a list of a string after a number under not-sorted",
				value:     `{"type":"list","items":[{"type":"int","value":1},{"type":"string","value":"a"}]}`,
				predicate: `{"kind":"not-sorted"}`,
			},
			{
				name:      "rejects an int under not-sorted",
				value:     oneLiteral,
				predicate: `{"kind":"not-sorted"}`,
			},
			{
				name:      "keeps a list of two equal elements under has-duplicate",
				value:     `{"type":"list","of":"int","value":[1,1]}`,
				predicate: `{"kind":"has-duplicate"}`,
				holds:     true,
			},
			{
				name:      "rejects a list of an int and a float of one number under has-duplicate",
				value:     `{"type":"list","items":[{"type":"int","value":1},{"type":"float","value":1.0}]}`,
				predicate: `{"kind":"has-duplicate"}`,
			},
			{
				name:      "rejects an int under has-duplicate",
				value:     oneLiteral,
				predicate: `{"kind":"has-duplicate"}`,
			},
			{
				name:      "keeps a list whose last element indexes a number above the n of indexed-above",
				value:     `{"type":"list","of":"int","value":[0,200,1]}`,
				predicate: `{"kind":"indexed-above","n":100}`,
				holds:     true,
			},
			{
				name:      "keeps a list whose last element 0 indexes a number above the n of indexed-above",
				value:     `{"type":"list","of":"int","value":[200,0]}`,
				predicate: `{"kind":"indexed-above","n":100}`,
				holds:     true,
			},
			{
				name:      "rejects a list whose last element indexes a number equal to the n of indexed-above",
				value:     `{"type":"list","of":"int","value":[0,100,1]}`,
				predicate: `{"kind":"indexed-above","n":100}`,
			},
			{
				name:      "rejects a list whose last element indexes itself under indexed-above",
				value:     `{"type":"list","of":"int","value":[5,1]}`,
				predicate: `{"kind":"indexed-above","n":0}`,
			},
			{
				name:      "rejects a list whose last element is negative under indexed-above",
				value:     `{"type":"list","of":"int","value":[200,-1]}`,
				predicate: `{"kind":"indexed-above","n":100}`,
			},
			{
				name:      "rejects a list whose last element is a float under indexed-above",
				value:     `{"type":"list","items":[{"type":"int","value":200},{"type":"float","value":0.0}]}`,
				predicate: `{"kind":"indexed-above","n":100}`,
			},
			{
				name:      "rejects a list whose last element is a bool under indexed-above",
				value:     `{"type":"list","items":[{"type":"int","value":200},{"type":"bool","value":false}]}`,
				predicate: `{"kind":"indexed-above","n":100}`,
			},
			{
				name:      "rejects a list whose last element indexes a bool under indexed-above",
				value:     `{"type":"list","items":[{"type":"bool","value":true},{"type":"int","value":0}]}`,
				predicate: `{"kind":"indexed-above","n":0}`,
			},
			{
				name:      "rejects an empty list under indexed-above",
				value:     `{"type":"list","of":"int","value":[]}`,
				predicate: `{"kind":"indexed-above","n":0}`,
			},
			{
				name:      "rejects a byte string under indexed-above",
				value:     `{"type":"bytes","value":"c800"}`,
				predicate: `{"kind":"indexed-above","n":100}`,
			},
			{
				name:      "rejects an int under indexed-above",
				value:     oneLiteral,
				predicate: `{"kind":"indexed-above","n":0}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Decoding, filtered(tt.value, tt.predicate, tt.holds)), nil, "")
			})
		}

		at := func(segs ...fault.Segment) fault.Path {
			return inVector(append([]fault.Segment{fault.Field(generatorAt), fault.Field("keep")}, segs...)...)
		}
		refusals := []struct {
			name       string
			predicate  string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault for a predicate that is no JSON object",
				predicate:  `3`,
				wantPath:   at(),
				wantReason: "the predicate does not parse",
			},
			{
				name:       "returns a fault at the kind for a predicate of an unknown kind",
				predicate:  `{"kind":"most"}`,
				wantPath:   at(fault.Field(kindAt)),
				wantReason: `"most" names no predicate`,
			},
			{
				name:       "returns a fault for equals without a value",
				predicate:  `{"kind":"equals"}`,
				wantPath:   at(),
				wantReason: "equals states no value",
			},
			{
				name:       "returns a fault at the value for contains of a literal of an unknown type",
				predicate:  `{"kind":"contains","value":` + widget + `}`,
				wantPath:   at(fault.Field(valueAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault for at-least without n",
				predicate:  `{"kind":"at-least"}`,
				wantPath:   at(),
				wantReason: "at-least states no n",
			},
			{
				name:       "returns a fault at n for at-least of n NaN",
				predicate:  `{"kind":"at-least","n":"NaN"}`,
				wantPath:   at(fault.Field("n")),
				wantReason: "NaN is no number of a predicate",
			},
			{
				name:       "returns a fault at n for at-least of an n of an unknown name",
				predicate:  `{"kind":"at-least","n":"Huge"}`,
				wantPath:   at(fault.Field("n")),
				wantReason: `"Huge" is none of the names NaN, Inf and -Inf`,
			},
			{
				name:       "returns a fault at n for at-least of an n that is no number",
				predicate:  `{"kind":"at-least","n":true}`,
				wantPath:   at(fault.Field("n")),
				wantReason: "the value is no float",
			},
			{
				name:       "returns a fault at n for divisible-by of a fractional n",
				predicate:  `{"kind":"divisible-by","n":2.5}`,
				wantPath:   at(fault.Field("n")),
				wantReason: "2.5 is no integer",
			},
			{
				name:       "returns a fault at n for length-at-least of a fractional n",
				predicate:  `{"kind":"length-at-least","n":1.5}`,
				wantPath:   at(fault.Field("n")),
				wantReason: "1.5 is no integer",
			},
			{
				name:       "returns a fault for indexed-above without n",
				predicate:  `{"kind":"indexed-above"}`,
				wantPath:   at(),
				wantReason: "indexed-above states no n",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := check(t, conformance.Decoding, filtered(oneLiteral, tt.predicate, true))
				expectFault(t, err, tt.wantPath, tt.wantReason)
			})
		}
	})
}

// filtered returns a decoding vector of a filter by predicate of the one
// typed literal value, which states that the case keeps the value when kept
// is true, and that the filter rejects the case otherwise.
func filtered(value, predicate string, kept bool) string {
	stated := null
	if kept {
		stated = value
	}
	filter := fmt.Sprintf(`{"gen":"filter","of":{"gen":"just","value":%s},"keep":%s}`, value, predicate)
	return decoded(filter, `[]`, `[]`, stated)
}
