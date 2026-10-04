// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// treeShape is a tree: a value and a list of children, each a tree.
const treeShape = `{"shape":"ref","name":"tree","definitions":{"tree":{"shape":"record","fields":[` +
	`["value",` + uint8Shape + `],["children",{"shape":"list","of":{"shape":"ref","name":"tree"}}]]}}}`

// recursiveTrips are recursive shapes, whose generated values each run back
// to choices that decode to them.
var recursiveTrips = []string{
	treeShape,
	`{"shape":"ref","name":"node","definitions":{"node":{"shape":"map","key":` + uint8Shape +
		`,"of":{"shape":"ref","name":"node"},"max_size":3}}}`,
	`{"shape":"ref","name":"node","definitions":{"node":{"shape":"enum","variants":[` +
		`["branch",{"shape":"set","of":{"shape":"ref","name":"node"},"max_size":2}],["leaf",` + uint8Shape + `]]}}}`,
}

// TestBudget checks the values of recursive shapes: the refs through their
// definitions, the budget of 100 nodes of one value, the exit that each
// container takes once a value has used it, and the inverse through a
// definition.
func TestBudget(t *testing.T) {
	t.Parallel()

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a tree through its definition", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, treeShape, n(3), n(1), n(4), n(0), n(0))
			want := record("value", uint64(3), "children", []any{record("value", uint64(4), "children", []any(nil))})
			assert.Equal(t, literal.Canonical(got), literal.Canonical(want), "a tree of one child with no children")
		})

		t.Run("returns a ref to a definition without recursion, which spends no budget", func(t *testing.T) {
			t.Parallel()
			got, _ := decode(t, `{"shape":"list","of":{"shape":"ref","name":"id"},"definitions":{"id":`+
				uint8Shape+`}}`, n(1), n(4), n(1), n(5), n(0))
			assert.Equal(t, got, any([]any{uint64(4), uint64(5)}), "the list of the definition's values")
		})

		t.Run("returns values that stop growing once they have used the budget", func(t *testing.T) {
			t.Parallel()
			g := read(t, treeShape)
			largest := 0
			for index := range uint64(300) {
				var v any
				engine.Generate(func(c *engine.Case) { v = engine.Draw(c, g, drawn) }, 5, index, nil)
				largest = max(largest, treeNodes(v))
			}
			assert.Equal(t, largest, 100, "the largest tree uses the whole budget and no more")
		})

		t.Run("returns the first exit of an enum once the budget is used", func(t *testing.T) {
			t.Parallel()
			nested := `{"shape":"ref","name":"node","definitions":{"node":{"shape":"enum","variants":[` +
				`["branch",{"shape":"fixed-list","size":2,"of":{"shape":"ref","name":"node"}}],["leaf",null]]}}}`
			zeros := make([]choice.Choice, 400)
			got, _ := decode(t, nested, zeros...)
			assert.True(t, variantNodes(got) < 300, "the branches end at leaves")
		})

		t.Run("returns a linked list that ends at an absent next", func(t *testing.T) {
			t.Parallel()
			linked := `{"shape":"ref","name":"node","definitions":{"node":{"shape":"record","fields":[` +
				`["value",` + uint8Shape + `],["next",{"shape":"optional","of":{"shape":"ref","name":"node"}}]]}}}`
			got, _ := decode(t, linked, n(1), n(1), n(2))
			want := record("value", uint64(1), "next", record("value", uint64(2), "next", nil))
			assert.Equal(t, got, any(want), "two nodes")
		})

		t.Run("returns values of definitions that refer to each other, cut at the budget", func(t *testing.T) {
			t.Parallel()
			mutual := `{"shape":"ref","name":"even","definitions":{` +
				`"even":{"shape":"optional","of":{"shape":"ref","name":"odd"}},` +
				`"odd":{"shape":"record","fields":[["next",{"shape":"ref","name":"even"}]]}}}`
			ones := make([]choice.Choice, 400)
			for i := range ones {
				ones[i] = n(1)
			}
			got, _ := decode(t, mutual, ones...)
			depth := 0
			for got != nil {
				got = got.(literal.Record).Fields[0].Value
				depth++
			}
			assert.Equal(t, depth, 50, "each odd node and its even node spend two of the budget's 100")
		})

		t.Run("returns maps that stop growing once they have used the budget", func(t *testing.T) {
			t.Parallel()
			keyed := `{"shape":"ref","name":"node","definitions":{"node":{"shape":"map",` +
				`"key":{"shape":"int","width":64,"signed":false},"of":{"shape":"ref","name":"node"}}}}`
			g := read(t, keyed)
			largest := 0
			for index := range uint64(200) {
				var v any
				engine.Generate(func(c *engine.Case) { v = engine.Draw(c, g, drawn) }, 3, index, nil)
				largest = max(largest, mapNodes(v))
			}
			assert.Equal(t, largest, 100, "the largest map uses the whole budget and no more")
		})
	})

	t.Run("Invert", func(t *testing.T) {
		t.Parallel()

		t.Run("returns choices that decode to each value that a recursive shape generates", func(t *testing.T) {
			t.Parallel()
			for _, text := range recursiveTrips {
				roundTrips(t, text)
			}
		})

		t.Run("returns a tree through its definition", func(t *testing.T) {
			t.Parallel()
			give := any(record("value", 3, "children", []any{record("value", 4, "children", []any{})}))
			got, err := engine.Invert(read(t, treeShape), give)
			assert.NoError(t, err, "the shape produces the tree")
			assert.True(t, slices.EqualFunc(got, []choice.Choice{n(3), n(1), n(4), n(0), n(0)}, choice.Choice.Equal),
				"the choices")
		})

		t.Run("returns an error for a tree past its budget, whose choices decode to another tree", func(t *testing.T) {
			t.Parallel()
			broad := make([]any, 101)
			for i := range broad {
				broad[i] = record("value", 1, "children", []any{})
			}
			_, err := engine.Invert(read(t, treeShape), any(record("value", 1, "children", broad)))
			assert.ErrorIs(t, err, engine.ErrCannotInvert, "no choices decode to the value")
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Nil(t, f.Path, "the tree as a whole")
			assert.HasPrefix(t, f.Reason, "the choices of ", "the replay decodes another tree")
		})
	})
}

// treeNodes returns the nodes of a tree of treeShape.
func treeNodes(v any) int {
	nodes := 1
	for _, child := range v.(literal.Record).Fields[1].Value.([]any) {
		nodes += treeNodes(child)
	}
	return nodes
}

// variantNodes returns the nodes of a value of branches and leaves.
func variantNodes(v any) int {
	variant := v.(literal.Variant)
	nodes := 1
	if variant.Name == "branch" {
		for _, child := range variant.Payload.([]any) {
			nodes += variantNodes(child)
		}
	}
	return nodes
}

// mapNodes returns the nodes of a value of maps of nodes.
func mapNodes(v any) int {
	nodes := 1
	for _, e := range v.(literal.Pairs).Entries {
		nodes += mapNodes(e.Value)
	}
	return nodes
}
