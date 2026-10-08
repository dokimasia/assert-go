// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestSpans checks the passes that delete, lift, target and reorder spans,
// each through a run from a stored failing case that the pass shrinks,
// pinned to what the definition's executable reference reports.
func TestSpans(t *testing.T) {
	t.Parallel()

	percent := engine.List(engine.Integer(0, 100), unbounded(t, 0))
	wide := engine.Integer(0, 1_000_000_000)
	digitTree := tree(t, 3, engine.DefaultMaxLeaves)
	firstAbove := func(c *engine.Case) string {
		values := engine.Draw(c, percent, "xs")
		return failsWhen(len(values) > 0 && values[0] > 10, "first")
	}

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			p      property
			stored []choice.Choice
			want   reference
		}{
			{
				name:   "deletes runs of siblings after a first element above 10",
				p:      firstAbove,
				stored: integers(1, 50, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{11}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEACwAA",
					runs:        22,
					calls:       24,
					digest:      "ae0a6e2dea05abfbfc6463ea6b5d30769c8eb900ef51de0203b4010de18ef026",
				},
			},
			{
				name:   "deletes three trailing siblings in one adaptive run",
				p:      firstAbove,
				stored: integers(1, 50, 1, 3, 1, 3, 1, 3, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{11}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEACwAA",
					runs:        21,
					calls:       23,
					digest:      "3d243fd7269c3910a2701ad80e9034affe6b3727fb9186da2723ffaeb5b8c686",
				},
			},
			{
				name: "deletes the last chunk of a list summing to 10 first",
				p: func(c *engine.Case) string {
					return failsWhen(sum(engine.Draw(c, percent, "xs")) >= 10, "ten")
				},
				stored: integers(1, 9, 1, 1, 1, 2, 1, 9, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{10}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEACgAA",
					runs:        41,
					calls:       43,
					digest:      "aedb0c920b08af45cf97924ee182c99119f52d50010c6ba0e4766c2fdb8f6d34",
				},
			},
			{
				name: "deletes the element before the one that fails",
				p: func(c *engine.Case) string {
					return failsWhen(slices.Contains(engine.Draw(c, percent, "xs"), 50), "fifty")
				},
				stored: integers(1, 3, 1, 50, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{50}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEAMgAA",
					runs:        19,
					calls:       21,
					digest:      "604bb6e165a1365252029de716b599d1c73a9191142c10360317ee87a30b284d",
				},
			},
			{
				name: "lifts the inner position of a tree that contains the failing leaf",
				p: func(c *engine.Case) string {
					return failsWhen(contains(engine.Draw(c, digitTree, "tree"), 9), "nine")
				},
				stored: integers(1, 1, 0, 9, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "tree", Value: 9, Relevance: engine.ValueMatters}},
					token:       "prop1:AAAACQ",
					runs:        13,
					calls:       15,
					digest:      "8c70ded41750a38784fdbbffff589fa7e910964d9faf5df024fc442d5982238f",
				},
			},
			{
				name: "deletes a sibling that the lift of a subtree lets go",
				p: func(c *engine.Case) string {
					value := engine.Draw(c, digitTree, "tree")
					_, list := value.([]any)
					return failsWhen(
						list && contains(value, 9) && (depth(value) < 2 || leafCount(value) >= 2),
						"lifted",
					)
				},
				stored: integers(1, 1, 1, 1, 0, 9, 1, 0, 1, 0, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "tree", Value: []any{9}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEAAQAAAAkAAA",
					runs:        40,
					calls:       42,
					digest:      "2d72cd8f1bb113a67ceb157d9927d8ab031c5c89738c5ad24c6e9281ce91b427",
				},
			},
			{
				name: "sets a draw that no failure depends on to its target",
				p: func(c *engine.Case) string {
					value := engine.Draw(c, wide, "n")
					engine.Draw(c, wide, "noise")
					return failsWhen(value > 1000, "above")
				},
				stored: integers(2000, 12345),
				want: reference{
					explanation: []engine.Explained{
						{Label: "n", Value: 1001, Relevance: engine.ValueMatters, NearestPassing: 1000},
						{Label: "noise", Value: 0, Relevance: engine.AnyValueFails},
					},
					token:  "prop1:AOkHAAA",
					runs:   41,
					calls:  43,
					digest: "3b0e8fbf90900a51c3b50da4f94ea7cfb110c815355ac51260da38f84892c2dd",
				},
			},
			{
				name: "puts the smaller of two siblings first",
				p: func(c *engine.Case) string {
					values := engine.Draw(c, percent, "xs")
					return failsWhen(slices.Contains(values, 3) && slices.Contains(values, 5), "both")
				},
				stored: integers(1, 5, 1, 3, 0),
				want: reference{
					explanation: []engine.Explained{{Label: "xs", Value: []int{3, 5}, Relevance: engine.ValueMatters}},
					token:       "prop1:AAEAAwABAAUAAA",
					runs:        48,
					calls:       50,
					digest:      "fcf8710ac01de5bf3216d2b74227db6e0eea7b3925de7f092b1c7d19cb90309b",
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
	})
}

// contains reports whether a tree of digits and lists contains wanted at
// any depth.
func contains(value any, wanted int) bool {
	if items, ok := value.([]any); ok {
		return slices.ContainsFunc(items, func(item any) bool { return contains(item, wanted) })
	}
	return value == wanted
}

// depth returns the number of lists around the deepest leaf of a tree of
// digits and lists: 0 for a leaf, and 1 for an empty list.
func depth(value any) int {
	items, ok := value.([]any)
	if !ok {
		return 0
	}
	deepest := 0
	for _, item := range items {
		deepest = max(deepest, depth(item))
	}
	return 1 + deepest
}
