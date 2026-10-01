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
					runs:   37,
					calls:  39,
					digest: "4cb124a385d949968ee7eab95438778199b1b347c88ffbf5a2ca1df188d66f9f",
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
					runs:        20,
					calls:       22,
					digest:      "866d86759712746d803937d16395910ef9ee4118a848e418c3999fd8ccc9e37e",
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
					runs:   32,
					calls:  34,
					digest: "f3e9717828bdb20cb2aa3f2047acd7bb38d2c6cdd4a5407d5f2951a35f8f4ee9",
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
					runs:   32,
					calls:  34,
					digest: "61db0b2f768edc8ed05002a4da289708a29e575e7aa9d592b524d4df6795db8c",
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
				runs:        76,
				calls:       95,
				digest:      "9605f9723f51bb09789ab85da54c76d7ec0c6d08a7e1837bc1f4e59974e37f37",
			})
			assert.Equal(t, got.Cases, 6, "the valid cases of seed 6 before the failure")
		})
	})
}
