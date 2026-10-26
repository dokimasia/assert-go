// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// recursiveAllocs is the ceiling of the allocations of Recursive: the
// engine's recursion, the closures of its decodes and of the inverses of a
// position and of the whole, the adapter of extend, the extension that
// extend builds, the generator of the position for a run that states the
// type, and the extension that decodes with it.
const recursiveAllocs = 20

// The types of the stated recursive generators, which only a run's
// generators produce.
type (
	// node is a node of a tree.
	node interface{ children() []node }
	// leaf is a node without children.
	leaf struct{}
	// branch is a node with children.
	branch struct{ kids []node }
	// fork is a struct with a field of node.
	fork struct {
		Next node `json:"next"`
	}
	// forest is a list of nodes.
	forest interface{ nodes() []node }
	// grove is a forest of its members.
	grove struct{ members []node }
)

func (leaf) children() []node     { return nil }
func (b branch) children() []node { return b.kids }
func (g grove) nodes() []node     { return g.members }

// TestRecursive checks the generator of values whose positions are values
// of the generator itself, and the bound on their base values.
func TestRecursive(t *testing.T) {
	t.Parallel()

	t.Run("MaxLeaves", func(t *testing.T) {
		t.Parallel()

		t.Run("makes every position past the bound take the base", func(t *testing.T) {
			t.Parallel()
			got := first(pairs(prop.MaxLeaves(1)), 1, 1, 0, 1, 1)
			assert.Equal(t, got, "(x x)", "the second position takes the base after one leaf")
		})

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { prop.MaxLeaves(0) }, "a bound of no leaf")
			assert.Equal(t, got, any("prop: MaxLeaves(0) is below 1"), "the panic names the option")
		})
	})

	t.Run("Recursive", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []prop.RecursiveOption
			want string
		}{
			{name: "extends a position whose choice is 1", want: "(x (x x))"},
			{name: "keeps the default bound after a zero option", give: []prop.RecursiveOption{{}}, want: "(x (x x))"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, first(pairs(tt.give...), 1, 1, 0, 1, 1), tt.want, "the nested pair")
			})
		}

		t.Run("returns the base's simplest value as the simplest", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, first(pairs()), "x", "the base")
		})

		t.Run("takes each value of a stated type inside its extension from its position", func(t *testing.T) {
			t.Parallel()
			levels := grown(t, prop.Using(prop.Recursive(prop.Just[node](leaf{}), branches)))
			assert.True(t, levels >= 3, "a branch inside a branch")
		})

		t.Run("takes the field of a stated type inside its extension from its position", func(t *testing.T) {
			t.Parallel()
			chains := prop.Recursive(prop.Just[node](leaf{}), func(prop.Generator[node]) prop.Generator[node] {
				return prop.Of[fork]().Map(func(f fork) node { return branch{[]node{f.Next}} })
			})
			assert.True(t, grown(t, prop.Using(chains)) >= 3, "a branch inside a branch")
		})

		t.Run("takes a pointer to a stated type inside its extension from its position", func(t *testing.T) {
			t.Parallel()
			chains := prop.Recursive(prop.Just[node](leaf{}), func(prop.Generator[node]) prop.Generator[node] {
				return prop.Of[*node]().Map(func(next *node) node {
					if next == nil {
						return branch{}
					}
					return branch{[]node{*next}}
				})
			})
			assert.True(t, grown(t, prop.Using(chains)) >= 3, "a branch inside a branch")
		})

		t.Run("takes the base at every position of a stated type past the bound", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			most := 0
			prop.ForAll(rec, contract, func(c *prop.Case) {
				leaves := 0
				for _, n := range preorder(nil, c.Draw(prop.Of[node](), drawn)) {
					if _, ok := n.(leaf); ok {
						leaves++
						continue
					}
					assert.True(c, leaves < 2, "a branch before the second leaf")
				}
				most = max(most, leaves)
			}, prop.Seed(7), prop.Using(prop.Recursive(prop.Just[node](leaf{}), branches, prop.MaxLeaves(2))))
			assert.False(t, rec.Failed(), "no branch follows the second leaf")
			assert.True(t, most > 2, "a value whose positions past the bound take the base")
		})

		t.Run("decodes a draw of a stated type inside its base without the stated generator", func(t *testing.T) {
			t.Parallel()
			inner := prop.Composite(func(c *prop.Case) node { return c.Draw(prop.Of[node](), "inner") })
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, func(c *prop.Case) { c.Draw(prop.Of[node](), drawn) },
				prop.Seed(7), prop.Using(prop.Recursive(inner, branches)))
			got, ok := rec.Failures()[0].Detail[failureField].(assert.Failure)
			assert.True(t, ok, "the failure of the case")
			assert.Equal(t, got.Contract,
				"prop: Of[prop_test.node]: node: prop_test.node has no variants, which RegisterVariants states",
				"the panic of the draw in the base")
		})

		t.Run("ends a cycle of two stated recursive generators", func(t *testing.T) {
			t.Parallel()
			nodes := prop.Recursive(prop.Just[node](leaf{}), func(prop.Generator[node]) prop.Generator[node] {
				return prop.Composite(func(c *prop.Case) node {
					return branch{c.Draw(prop.Of[forest](), "forest").nodes()}
				})
			})
			forests := prop.Recursive(prop.Just[forest](grove{}), func(prop.Generator[forest]) prop.Generator[forest] {
				return prop.Composite(func(c *prop.Case) forest {
					return grove{c.Draw(prop.Of[[]node](), "members")}
				})
			})
			assert.True(t, grown(t, prop.Using(nodes), prop.Using(forests)) >= 3, "a node of a forest of a node")
		})
	})
}

// branches returns the generator of a branch whose children a draw of Of
// decodes.
func branches(prop.Generator[node]) prop.Generator[node] {
	return prop.Composite(func(c *prop.Case) node {
		return branch{c.Draw(prop.Of[[]node](), "children")}
	})
}

// grown runs a property that draws a node in each case under opts, checks
// that the run passes, and returns the most levels of one node.
func grown(t *testing.T, opts ...prop.Option) int {
	t.Helper()
	rec := assert.NewRecorder()
	most := 0
	prop.ForAll(rec, contract, func(c *prop.Case) {
		most = max(most, levelsOf(c.Draw(prop.Of[node](), drawn)))
	}, append([]prop.Option{prop.Seed(7)}, opts...)...)
	assert.False(t, rec.Failed(), "every case draws a node")
	return most
}

// levelsOf returns the number of nodes on the longest path from n to a node
// without children.
func levelsOf(n node) int {
	most := 0
	for _, child := range n.children() {
		most = max(most, levelsOf(child))
	}
	return most + 1
}

// preorder appends the nodes of n to out in the order that a decode
// chooses them: n, then the nodes of each child in order.
func preorder(out []node, n node) []node {
	out = append(out, n)
	for _, child := range n.children() {
		out = preorder(out, child)
	}
	return out
}

// TestRecursiveAllocs checks the allocation ceilings of the recursive
// generator and its option.
func TestRecursiveAllocs(t *testing.T) {
	base := prop.Just("x")
	assert.MaxAllocs(t, func() { _ = prop.MaxLeaves(1) }, 0, "MaxLeaves allocates nothing")
	assert.MaxAllocs(t, func() { _ = prop.Recursive(base, nest) }, recursiveAllocs,
		"Recursive allocates its recursion and its decodes")
}

// BenchmarkRecursive measures the recursive generator and its option.
func BenchmarkRecursive(b *testing.B) {
	b.Run("MaxLeaves", func(b *testing.B) {
		var got prop.RecursiveOption
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.MaxLeaves(1)
		}
		assert.Equal(b, got, prop.MaxLeaves(1), "the option")
	})

	b.Run("Recursive", func(b *testing.B) {
		var got prop.Generator[string]
		base := prop.Just("x")
		c := bench.Start(b).MaxAllocs(recursiveAllocs)
		defer c.End()
		for c.Loop() {
			got = prop.Recursive(base, nest)
		}
		assert.Equal(b, first(got, 1, 0), "(x)", "the nested value")
	})
}

// pairs returns the generator of "x", or of a pair of its own values in
// parentheses, under opts.
func pairs(opts ...prop.RecursiveOption) prop.Generator[string] {
	pair := func(self prop.Generator[string]) prop.Generator[string] {
		return prop.List(self, prop.MinSize(2), prop.MaxSize(2)).Map(func(p []string) string {
			return "(" + p[0] + " " + p[1] + ")"
		})
	}
	return prop.Recursive(prop.Just("x"), pair, opts...)
}

// nest returns the generator of a value of self in parentheses.
func nest(self prop.Generator[string]) prop.Generator[string] {
	return self.Map(func(s string) string { return "(" + s + ")" })
}
