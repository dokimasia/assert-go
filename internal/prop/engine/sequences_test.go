// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestSequences checks the passes that delete and lower the elements of
// sequences and delete pairs of structure choices, each through a run that
// the pass shrinks, pinned to what the definition's executable reference
// reports.
func TestSequences(t *testing.T) {
	t.Parallel()

	digit, wide := engine.Integer(0, 9), engine.Integer(0, 1_000_000_000)
	nested := engine.List(engine.List(digit, unbounded(t, 0)), unbounded(t, 0))
	pair := engine.List(digit, sizes(t, 2, 2))
	ones := engine.List(engine.List(digit, sizes(t, 1, 1)), sizes(t, 2, 2))
	bytes, single := engine.Bytes(unbounded(t, 0)), engine.Bytes(sizes(t, 1, 1))
	high := func(g engine.Generator[[]byte]) property {
		return func(c *engine.Case) string {
			return failsWhen(slices.ContainsFunc(engine.Draw(c, g, "b"), func(b byte) bool { return b > 200 }), "high")
		}
	}
	nine := func(inner []int) bool { return slices.Contains(inner, 9) }

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			p      property
			stored []choice.Choice
			want   reference
		}{
			{
				name: "joins two lists by deleting a stop flag and a continue flag",
				p: func(c *engine.Case) string {
					distinct := map[int]bool{}
					for _, inner := range engine.Draw(c, nested, "xss") {
						for _, v := range inner {
							distinct[v] = true
						}
					}
					return failsWhen(len(distinct) >= 2, "mixed")
				},
				stored: integers(1, 1, 1, 0, 1, 1, 2, 0, 0),
				want: reference{
					explanation: []engine.Explained{
						{Label: "xss", Value: [][]int{{0, 1}}, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AAEAAQAAAAEAAQAAAAA",
					runs:   39,
					calls:  41,
					digest: "e7d0957d55c35614c83752eee352b21d81c8bc46a1a0ebe5789d400df353a3d7",
				},
			},
			{
				name: "leaves the forced flags of a list of one length",
				p: func(c *engine.Case) string {
					return failsWhen(slices.Contains(engine.Draw(c, pair, "xs"), 9), "nine")
				},
				stored: integers(1, 9, 1, 9, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{0, 9}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEAAAABAAkAAA",
					runs:        22,
					calls:       24,
					digest:      "ed2709e82a77aa82474dd0becc3efc32f265313e6efdd62884352a90ba6a01e0",
				},
			},
			{
				name: "leaves the forced flags of lists of one length in a list of one length",
				p: func(c *engine.Case) string {
					return failsWhen(slices.ContainsFunc(engine.Draw(c, ones, "xss"), nine), "nine")
				},
				stored: integers(1, 1, 9, 0, 1, 1, 9, 0, 0),
				want: reference{
					explanation: []engine.Explained{
						{Label: "xss", Value: [][]int{{0}, {9}}, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AAEAAQAAAAAAAQABAAkAAAAA",
					runs:   36,
					calls:  38,
					digest: "201f60ead739fa0baba6287c47ad6ac15269079623e4ffa202170bfbadcc63a2",
				},
			},
			{
				name:   "deletes the bytes around a high byte and lowers it to 201",
				p:      high(bytes),
				stored: []choice.Choice{sequence(1, 250, 3, 4)},
				want: reference{
					explanation: []engine.Explained{{Label: "b", Value: []byte{201}, Relevance: engine.ValueMatters}},
					token:       "prop1:AwHJAQ",
					runs:        21,
					calls:       23,
					digest:      "c40b06aa4ff84ac32ef03026788604899da602be5ecdc42bf2e56f0fb65e387a",
				},
			},
			{
				name:   "deletes no byte below the minimum length",
				p:      high(engine.Bytes(unbounded(t, 1))),
				stored: []choice.Choice{sequence(1, 250)},
				want: reference{
					explanation: []engine.Explained{{Label: "b", Value: []byte{201}, Relevance: engine.ValueMatters}},
					token:       "prop1:AwHJAQ",
					runs:        20,
					calls:       22,
					digest:      "abeb8cbf5e32058efcbeebd72d20f7f0a08c1bfa631e1314829d789fa7b3df9e",
				},
			},
			{
				name: "lowers the first byte to 0 when the second one fails",
				p: func(c *engine.Case) string {
					b := engine.Draw(c, bytes, "b")
					return failsWhen(len(b) >= 2 && b[1] > 100, "second")
				},
				stored: []choice.Choice{sequence(50, 200)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "b", Value: []byte{0, 101}, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AwIAZQ",
					runs:   23,
					calls:  25,
					digest: "2ec34ff75075cd59b5b3665ee2ac4fbfc56418a9426cec22f0322cdb26e9e25d",
				},
			},
			{
				name: "lowers a byte that counts the digits after it to 2",
				p: func(c *engine.Case) string {
					count := engine.Draw(c, single, "count")[0]
					for range count {
						engine.Draw(c, digit, "digit")
					}
					return failsWhen(count >= 2, "many")
				},
				stored: append([]choice.Choice{sequence(3)}, integers(0, 0, 0)...),
				want: reference{
					explanation: []engine.Explained{
						{Label: "count", Value: []byte{2}, Relevance: engine.AnyValueFails},
						{Label: "digit", Value: 0, Relevance: engine.AnyValueFails},
						{Label: "digit", Value: 0, Relevance: engine.AnyValueFails},
					},
					token:  "prop1:AwECAAAAAA",
					runs:   25,
					calls:  27,
					digest: "39da6951292e4f892be9590566f30186c001219ebf9580943afcb41262a1979a",
				},
			},
			{
				name: "leaves an empty byte string that no failure depends on",
				p: func(c *engine.Case) string {
					engine.Draw(c, bytes, "b")
					return failsWhen(engine.Draw(c, wide, "n") > 1000, "above")
				},
				stored: []choice.Choice{sequence(), unsigned(2000)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "b", Value: []byte{}, Relevance: engine.AnyValueFails},
						{Label: "n", Value: 1001, Relevance: engine.ValueMatters, NearestPassing: 1000},
					},
					token:  "prop1:AwAA6Qc",
					runs:   33,
					calls:  35,
					digest: "e0d06f0aeb123b1e919e7279b22f721edd74b2ef361fdec39507baa32806e60d",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, trace := traced(tt.p, settled(tt.stored...))
				matchesReference(t, got, trace, tt.want)
			})
		}

		t.Run("deletes a stop flag and a continue flag of a tree with three leaves", func(t *testing.T) {
			t.Parallel()
			s := settled()
			s.Seed = 6
			digitTree := tree(t, 3, engine.DefaultMaxLeaves)
			got, trace := recorded(func(c *engine.Case) {
				if leafCount(engine.Draw(c, digitTree, "tree")) >= 3 {
					c.Report(assert.Failure{Assertion: "three"}, false)
				}
			}, s)
			matchesReference(t, got, trace, reference{
				explanation: []engine.Explained{{Label: "tree", Value: []any{0, 0, 0}, Relevance: engine.ValueMatters}},
				token:       "prop1:AAEAAQAAAAAAAQAAAAAAAQAAAAAAAA",
				runs:        78,
				calls:       97,
				digest:      "dcd99fc394fab6903e0071d115f271ca99426dcc79a4bf150af6658e45b9bd48",
			})
			assert.Equal(t, got.Cases, 6, "the valid cases of seed 6 before the failure")
		})
	})
}
