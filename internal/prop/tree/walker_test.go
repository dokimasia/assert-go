// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package tree_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/tree"
)

// TestWalker checks the repeats and the divergences that a walk finds, and
// what it records at the limit of its tree.
func TestWalker(t *testing.T) {
	t.Parallel()

	digit, bit := integerBounds(t, 9), integerBounds(t, 1)

	t.Run("Restart", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the walker to the root of its tree", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			w := tr.Walk()
			assert.NoError(t, w.Step(digit, integer(7)), "the first case of 7")
			assert.NoError(t, w.End(), "the end of the first case")
			w.Restart()
			assert.ErrorIs(t, w.Step(digit, integer(7)), tree.ErrRepeated, "the second case of 7, from the root")
		})

		t.Run("checks the steps of a case after a walk that found the tree at its limit", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(3)
			assert.NoError(t, walk(tr, step{bit, integer(0)}, step{bit, integer(0)}), "a case that fills the tree")
			w := tr.Walk()
			assert.NoError(t, w.Step(bit, integer(0)), "a recorded choice")
			assert.NoError(t, w.Step(bit, integer(1)), "a choice past the limit")
			w.Restart()
			assert.NoError(t, w.Step(bit, integer(0)), "a recorded choice of the next case")
			assert.ErrorIs(t, w.Step(bit, integer(0)), tree.ErrRepeated, "the repeat that the next case finds")
		})
	})

	t.Run("Step", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrRepeated for a choice that arrives at a leaf", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			assert.NoError(t, walk(tr, step{digit, integer(7)}), "the first case of 7")
			assert.ErrorIs(t, walk(tr, step{digit, integer(7)}), tree.ErrRepeated, "the second case of 7")
		})

		t.Run("returns ErrRepeated for a NaN of another sign and payload", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			nan := floatBounds(t, math.Inf(-1), math.Inf(1), choice.AdmitNaN)
			assert.NoError(t, walk(tr, step{nan, float(math.NaN())}), "the first NaN")
			other := float(math.Float64frombits(0xFFF8000000000001))
			assert.ErrorIs(t, walk(tr, step{nan, other}), tree.ErrRepeated, "every NaN is one value")
		})

		t.Run("returns nil for -0 after +0", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			reals := floatBounds(t, -1, 1, choice.ExcludeNaN)
			assert.NoError(t, walk(tr, step{reals, float(0)}), "+0")
			assert.NoError(t, walk(tr, step{reals, float(math.Copysign(0, -1))}), "-0 is another value")
			assert.Equal(t, tr.Nodes(), 3, "the root and two leaves")
		})

		t.Run("returns ErrRepeated for a sequence with the same elements", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			bytes := sequenceBounds(t, 256, 0, 4)
			for _, s := range [][]uint32{{1, 2}, {2, 1}, {1}, {}} {
				assert.NoError(t, walk(tr, step{bytes, sequence(s...)}), "a new sequence")
			}
			assert.ErrorIs(t, walk(tr, step{bytes, sequence(1, 2)}), tree.ErrRepeated, "the same elements")
		})

		t.Run("returns a DivergenceError for other bounds at a recorded position", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			assert.NoError(t, walk(tr, step{digit, integer(1)}, step{digit, integer(2)}), "the first case")
			err := walk(tr, step{digit, integer(1)}, step{bit, integer(0)})
			diverged := assert.ErrorAs[*tree.DivergenceError](t, err, "the divergence")
			assert.Equal(t, diverged.Index, 1, "the position")
			assert.Equal(t, *diverged.Recorded, digit, "the recorded bounds")
			assert.Equal(t, *diverged.Requested, bit, "the requested bounds")
		})

		t.Run("returns a DivergenceError for a request where an earlier case ended", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			assert.NoError(t, walk(tr), "a case without choices")
			diverged := assert.ErrorAs[*tree.DivergenceError](t, walk(tr, step{digit, integer(1)}), "the divergence")
			assert.Equal(t, diverged.Index, 0, "the position")
			assert.Nil(t, diverged.Recorded, "an end is recorded there")
			assert.Equal(t, *diverged.Requested, digit, "the requested bounds")
		})

		t.Run("returns nil for the rest of a case that found the tree at its limit", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(3)
			assert.NoError(t, walk(tr, step{bit, integer(0)}, step{bit, integer(0)}), "a case that fills the tree")
			w := tr.Walk()
			assert.NoError(t, w.Step(bit, integer(0)), "a recorded choice")
			assert.NoError(t, w.Step(bit, integer(1)), "a choice past the limit")
			assert.NoError(t, w.Step(digit, integer(5)), "a choice that the walk no longer checks")
			assert.NoError(t, w.End(), "an end that the walk no longer records")
			assert.Equal(t, tr.Nodes(), 3, "the nodes of the full tree")
		})
	})

	t.Run("End", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a DivergenceError for an end where an earlier case made a request", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			assert.NoError(t, walk(tr, step{digit, integer(1)}), "a case with one choice")
			diverged := assert.ErrorAs[*tree.DivergenceError](t, walk(tr), "the divergence")
			assert.Equal(t, diverged.Index, 0, "the position")
			assert.Equal(t, *diverged.Recorded, digit, "the recorded bounds")
			assert.Nil(t, diverged.Requested, "this case ends there")
		})

		t.Run("returns nil for a second end at a root leaf", func(t *testing.T) {
			t.Parallel()
			tr := tree.New(tree.NodeLimit)
			assert.NoError(t, walk(tr), "a case without choices")
			assert.NoError(t, walk(tr), "the same case again")
			assert.True(t, tr.Exhausted(), "the root is still a leaf")
		})
	})
}

// TestWalkerAllocs checks that a restart, a step on an existing edge and
// an end at an existing leaf allocate nothing.
func TestWalkerAllocs(t *testing.T) {
	digit := integerBounds(t, 9)
	restarted := tree.New(tree.NodeLimit).Walk()
	assert.MaxAllocs(t, restarted.Restart, 0, "Restart allocates nothing")
	repeated := tree.New(tree.NodeLimit)
	assert.NoError(t, walk(repeated, step{digit, integer(7)}), "a case of 7")
	at := repeated.Walk()
	assert.MaxAllocs(t, func() { _ = at.Step(digit, integer(7)) }, 0, "Step on an existing edge allocates nothing")
	empty := tree.New(tree.NodeLimit)
	assert.NoError(t, walk(empty), "a case without choices")
	again := empty.Walk()
	assert.MaxAllocs(t, func() { _ = again.End() }, 0, "End at an existing leaf allocates nothing")
}

// BenchmarkWalker measures a restart, a step on an existing edge and an end
// at an existing leaf under a ceiling of no allocation.
func BenchmarkWalker(b *testing.B) {
	digit := integerBounds(b, 9)

	b.Run("Restart", func(b *testing.B) {
		tr := tree.New(tree.NodeLimit)
		assert.NoError(b, walk(tr, step{digit, integer(7)}), "a case of 7")
		w := tr.Walk()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			w.Restart()
		}
		assert.ErrorIs(b, w.Step(digit, integer(7)), tree.ErrRepeated, "the walker is at the root")
	})

	b.Run("Step", func(b *testing.B) {
		var got error
		tr := tree.New(tree.NodeLimit)
		assert.NoError(b, walk(tr, step{digit, integer(7)}), "a case of 7")
		w := tr.Walk()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = w.Step(digit, integer(7))
		}
		assert.ErrorIs(b, got, tree.ErrRepeated, "a repeated case")
	})

	b.Run("End", func(b *testing.B) {
		var got error
		tr := tree.New(tree.NodeLimit)
		assert.NoError(b, walk(tr), "a case without choices")
		w := tr.Walk()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = w.End()
		}
		assert.NoError(b, got, "an end at the root leaf")
	})
}
