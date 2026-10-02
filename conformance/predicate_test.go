// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// The literals that the predicates are tested on.
const (
	// oneLiteral is the int 1.
	oneLiteral = `{"type":"int","value":1}`
	// pairLiteral is the list of the ints 1 and 2.
	pairLiteral = `{"type":"list","of":"int","value":[1,2]}`
)

// TestPredicate checks each predicate of the vocabulary through a filter of
// one stated value: the case keeps the value where the predicate reports
// true, and is rejected where it reports false.
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
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Decoding, filtered(tt.value, tt.predicate, tt.holds)), "")
			})
		}

		refusals := []struct {
			name      string
			predicate string
			want      string
		}{
			{
				name:      "returns an error for a predicate that is no JSON object",
				predicate: `3`,
				want:      "parse predicate",
			},
			{
				name:      "returns an error for a predicate of an unknown kind",
				predicate: `{"kind":"most"}`,
				want:      `"most" names no predicate`,
			},
			{
				name:      "returns an error for equals without a value",
				predicate: `{"kind":"equals"}`,
				want:      "predicate equals lacks value",
			},
			{
				name:      "returns an error for contains of a literal of an unknown type",
				predicate: `{"kind":"contains","value":{"type":"widget"}}`,
				want:      conformance.ErrUnknownType.Error(),
			},
			{
				name:      "returns an error for at-least without n",
				predicate: `{"kind":"at-least"}`,
				want:      "predicate at-least lacks n",
			},
			{
				name:      "returns an error for at-least of n NaN",
				predicate: `{"kind":"at-least","n":"NaN"}`,
				want:      "the number of a predicate is NaN",
			},
			{
				name:      "returns an error for at-least of an n of an unknown name",
				predicate: `{"kind":"at-least","n":"Huge"}`,
				want:      `unrecognized float literal "Huge"`,
			},
			{
				name:      "returns an error for at-least of an n that is no number",
				predicate: `{"kind":"at-least","n":true}`,
				want:      "decode float",
			},
			{
				name:      "returns an error for divisible-by of a fractional n",
				predicate: `{"kind":"divisible-by","n":2.5}`,
				want:      "predicate divisible-by states 2.5, no integer",
			},
			{
				name:      "returns an error for length-at-least of a fractional n",
				predicate: `{"kind":"length-at-least","n":1.5}`,
				want:      "predicate length-at-least states 1.5, no integer",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Decoding, filtered(oneLiteral, tt.predicate, true)), tt.want)
			})
		}
	})
}

// filtered returns a decoding vector of a filter by predicate of the one
// typed literal value, which states that the case keeps the value where
// holds is set, and that the filter rejects the case where not.
func filtered(value, predicate string, holds bool) string {
	stated := null
	if holds {
		stated = value
	}
	filter := fmt.Sprintf(`{"gen":"filter","of":{"gen":"just","value":%s},"keep":%s}`, value, predicate)
	return decoded(filter, `[]`, `[]`, stated)
}
