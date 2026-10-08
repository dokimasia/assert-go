// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestSpan checks what a span states about the choices of one generator:
// their range, the spans open around it, and its parent.
func TestSpan(t *testing.T) {
	t.Parallel()

	t.Run("Spans", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a span whose end is its start for a generator that made no choice", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, engine.Just(5))
			assert.Equal(t, e.Case.Spans(), []engine.Span{{Label: "just", Start: 0, End: 0, Depth: 0, Parent: -1}},
				"an empty span at the top")
		})

		t.Run("returns the depth and the parent of each nested span", func(t *testing.T) {
			t.Parallel()
			g := engine.Optional(engine.Integer(0, 9).Bind(engine.Just[int]))
			_, e := decode(t, g, integers(1, 4)...)
			assert.Equal(t, e.Case.Spans(), []engine.Span{
				{Label: "optional", Start: 0, End: 2, Depth: 0, Parent: -1},
				{Label: "bind", Start: 1, End: 2, Depth: 1, Parent: 0},
				{Label: "integer", Start: 1, End: 2, Depth: 2, Parent: 1},
				{Label: "just", Start: 2, End: 2, Depth: 2, Parent: 1},
			}, "each span inside the one opened before it")
		})

		t.Run("returns spans at the top for two draws", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				engine.Draw(c, engine.Integer(0, 9), "first")
				engine.Draw(c, engine.Integer(0, 9), "second")
			}, integers(3, 4), nil)
			assert.Equal(t, e.Case.Spans(), []engine.Span{
				{Label: "integer", Start: 0, End: 1, Depth: 0, Parent: -1},
				{Label: "integer", Start: 1, End: 2, Depth: 0, Parent: -1},
			}, "two spans without a parent")
		})
	})
}
