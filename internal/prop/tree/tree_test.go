// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package tree_test

import (
	"math"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/tree"
)

// pinnedNodeLimit is the node limit of a run's tree, as the definition
// states it.
const pinnedNodeLimit = 1 << 20

// TestTree checks the size of a tree, its limit, and when its domain is
// exhausted.
func TestTree(t *testing.T) {
	t.Parallel()

	bit := integerBounds(t, 1)

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a tree of the root alone", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tree.New(tree.NodeLimit).Nodes(), 1, "the root")
		})

		t.Run("returns a run's tree at a limit of 2^20 nodes", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tree.NodeLimit, pinnedNodeLimit, "the limit of a run's tree")
		})
	})

	t.Run("Exhausted", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true once every value of two bits reached a leaf", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			for _, c := range [][2]int64{{0, 0}, {0, 1}, {1, 0}} {
				assert.NoError(t, walk(tr, step{bit, integer(c[0])}, step{bit, integer(c[1])}), "a new case")
				assert.False(t, tr.Exhausted(), "a value is still untested")
			}
			assert.NoError(t, walk(tr, step{bit, integer(1)}, step{bit, integer(1)}), "the last case")
			assert.True(t, tr.Exhausted(), "every case is tested")
		})

		t.Run("reports true for a shorter branch that its leaf exhausts", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			assert.NoError(t, walk(tr, step{bit, integer(0)}), "a 0 ends at once")
			assert.NoError(t, walk(tr, step{bit, integer(1)}, step{bit, integer(0)}), "a 1 then a 0")
			assert.False(t, tr.Exhausted(), "a 1 then a 1 is untested")
			assert.NoError(t, walk(tr, step{bit, integer(1)}, step{bit, integer(1)}), "a 1 then a 1")
			assert.True(t, tr.Exhausted(), "every case is tested")
		})

		t.Run("reports true after a case without choices", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			assert.NoError(t, walk(tr), "a case without choices")
			assert.True(t, tr.Exhausted(), "the root is the leaf")
		})

		t.Run("reports true after every digit", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			digit := integerBounds(t, 9)
			for v := range int64(10) {
				assert.False(t, tr.Exhausted(), "a digit is untested before "+strconv.FormatInt(v, 10))
				assert.NoError(t, walk(tr, step{digit, integer(v)}), "a new digit")
			}
			assert.True(t, tr.Exhausted(), "every digit is tested")
		})

		t.Run("reports true after the seven sequences of up to two bits", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			bits := sequenceBounds(t, 2, 0, 2)
			for _, s := range [][]uint32{{}, {0}, {1}, {0, 0}, {0, 1}, {1, 0}} {
				assert.NoError(t, walk(tr, step{bits, sequence(s...)}), "a new sequence")
			}
			assert.False(t, tr.Exhausted(), "one sequence is untested")
			assert.NoError(t, walk(tr, step{bits, sequence(1, 1)}), "the last sequence")
			assert.True(t, tr.Exhausted(), "every sequence is tested")
		})

		t.Run("reports true after the three sequences of zeros from length 1 to 3", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			zeros := sequenceBounds(t, 1, 1, 3)
			assert.NoError(t, walk(tr, step{zeros, sequence(0)}), "one zero")
			assert.NoError(t, walk(tr, step{zeros, sequence(0, 0)}), "two zeros")
			assert.False(t, tr.Exhausted(), "three zeros are untested")
			assert.NoError(t, walk(tr, step{zeros, sequence(0, 0, 0)}), "three zeros")
			assert.True(t, tr.Exhausted(), "every sequence is tested")
		})

		tests := []struct {
			name string
			step step
		}{
			{
				name: "reports false for a float of one value",
				step: step{floatBounds(t, 1, 1, choice.ExcludeNaN), float(1)},
			},
			{
				name: "reports false for a sequence without a maximum length",
				step: step{unboundedSequenceBounds(t, 1), sequence()},
			},
			{
				name: "reports false for 2^64 sequences or more",
				step: step{sequenceBounds(t, 1<<31, 0, 3), sequence()},
			},
			{
				name: "reports false for the whole unsigned range",
				step: step{unsignedBounds(t), integer(0)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				tr := tree.New(tree.NodeLimit)
				assert.NoError(t, walk(tr, tt.step), "a new case")
				assert.False(t, tr.Exhausted(), "a domain that the tree never exhausts")
			})
		}
	})

	t.Run("Full", func(t *testing.T) {
		t.Parallel()

		t.Run("reports false below the limit", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(3)
			assert.NoError(t, walk(tr, step{bit, integer(0)}, step{bit, integer(0)}), "a case of three nodes")
			assert.False(t, tr.Full(), "no walk went past the limit")
		})

		t.Run("reports true once a walk found the tree at its limit", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(3)
			assert.NoError(t, walk(tr, step{bit, integer(0)}, step{bit, integer(0)}), "a case of three nodes")
			assert.NoError(t, walk(tr, step{bit, integer(0)}, step{bit, integer(1)}), "a case past the limit")
			assert.True(t, tr.Full(), "the tree is at its limit")
			assert.False(t, tr.Exhausted(), "a full tree reports no exhausted domain")
		})
	})

	t.Run("Nodes", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the root and one node per distinct value", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			assert.NoError(t, walk(tr, step{bit, integer(0)}, step{bit, integer(1)}), "two choices")
			assert.NoError(t, walk(tr, step{bit, integer(0)}, step{bit, integer(0)}), "one more value")
			assert.Equal(t, tr.Nodes(), 4, "the root, the 0 and its two children")
		})
	})

	t.Run("Walk", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a walker at the root", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			w := tr.Walk()
			assert.NoError(t, w.Step(bit, integer(1)), "a choice at the root")
			assert.Equal(t, tr.Nodes(), 2, "the root and its child")
		})
	})
}

// TestTreeAllocs checks that the queries of a tree allocate nothing.
func TestTreeAllocs(t *testing.T) {
	tr := tree.New(tree.NodeLimit)
	assert.MaxAllocs(t, func() { _ = tr.Exhausted() }, 0, "Exhausted allocates nothing")
	assert.MaxAllocs(t, func() { _ = tr.Full() }, 0, "Full allocates nothing")
	assert.MaxAllocs(t, func() { _ = tr.Nodes() }, 0, "Nodes allocates nothing")
}

// BenchmarkTree measures the constructor, the queries and a new walker. New
// allocates the tree, its node slice and its edge map, and Walk the walker
// and its path.
func BenchmarkTree(b *testing.B) {
	tr := tree.New(tree.NodeLimit)

	b.Run("New", func(b *testing.B) {
		var got *tree.Tree
		c := bench.Start(b).MaxAllocs(3)
		defer c.End()
		for c.Loop() {
			got = tree.New(tree.NodeLimit)
		}
		assert.Equal(b, got.Nodes(), 1, "the root")
	})

	b.Run("Exhausted", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = tr.Exhausted()
		}
		assert.False(b, got, "an empty tree tested nothing")
	})

	b.Run("Full", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = tr.Full()
		}
		assert.False(b, got, "an empty tree is below its limit")
	})

	b.Run("Nodes", func(b *testing.B) {
		var got int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = tr.Nodes()
		}
		assert.Equal(b, got, 1, "the root")
	})

	b.Run("Walk", func(b *testing.B) {
		var got *tree.Walker
		c := bench.Start(b).MaxAllocs(2)
		defer c.End()
		for c.Loop() {
			got = tr.Walk()
		}
		assert.NotNil(b, got, "a walker")
	})
}

// unboundedSequenceBounds returns the bounds of a sequence of integers in
// [0, k) of any length, failing the test when they are invalid.
func unboundedSequenceBounds(tb testing.TB, k uint32) choice.Bounds {
	tb.Helper()
	sizes, err := choice.NewUnboundedSizes(0)
	assert.NoError(tb, err, "the sizes are valid")
	b, err := choice.NewSequenceBounds(k, sizes)
	assert.NoError(tb, err, "the bounds are valid")
	return choice.OfSequence(b)
}

// unsignedBounds returns the bounds of the whole unsigned 64-bit range.
func unsignedBounds(tb testing.TB) choice.Bounds {
	tb.Helper()
	b, err := choice.NewIntegerBounds(choice.Int{}, choice.UintOf(math.MaxUint64))
	assert.NoError(tb, err, "the bounds are valid")
	return choice.OfInteger(b)
}
