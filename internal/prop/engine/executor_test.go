// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"sync/atomic"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestExecutor checks where the cases of the generation phase meet the
// case tree: as they run on one worker, and after they ran on more.
func TestExecutor(t *testing.T) {
	t.Parallel()

	digit := engine.Integer(0, 9)

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("stops a repeated case at its repeat on one worker", func(t *testing.T) {
			t.Parallel()
			var ended atomic.Int64
			s := settled()
			s.Workers = 1
			engine.Run(func(c *engine.Case) {
				engine.Draw(c, digit, drawn)
				ended.Add(1)
			}, s)
			assert.Equal(t, ended.Load(), int64(10), "one body for each digit runs past its draw")
		})

		t.Run("runs a repeated case to its end on more workers", func(t *testing.T) {
			t.Parallel()
			var ended atomic.Int64
			s := settled()
			s.Workers = fourWorkers
			engine.Run(func(c *engine.Case) {
				engine.Draw(c, digit, drawn)
				ended.Add(1)
			}, s)
			assert.True(t, ended.Load() > 10, "the repeats run past their draws too")
		})
	})
}
