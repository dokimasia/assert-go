// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tree_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/tree"
)

// TestDivergenceError checks the text of a divergence for each way two
// cases part.
func TestDivergenceError(t *testing.T) {
	t.Parallel()

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		digit := integerBounds(t, 9)
		tests := []struct {
			name string
			give tree.DivergenceError
			want string
		}{
			{
				name: "returns the position where an earlier case ended",
				give: tree.DivergenceError{Index: 2, Requested: &digit},
				want: "tree: the body diverged at choice 2, where an earlier case ended",
			},
			{
				name: "returns the position where this case ended",
				give: tree.DivergenceError{Index: 0, Recorded: &digit},
				want: "tree: the body diverged at choice 0, where this case ended",
			},
			{
				name: "returns the position where the case requested other bounds",
				give: tree.DivergenceError{Index: 1, Recorded: &digit, Requested: &digit},
				want: "tree: the body diverged at choice 1, where it requested other bounds",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Error(), tt.want, "the text of the divergence")
			})
		}
	})
}

// BenchmarkDivergenceError measures the text of a divergence, which
// allocates the number and the joined text.
func BenchmarkDivergenceError(b *testing.B) {
	b.Run("Error", func(b *testing.B) {
		var got string
		digit := choice.Bounds{}
		d := &tree.DivergenceError{Index: 12, Recorded: &digit, Requested: &digit}
		c := bench.Start(b).MaxAllocs(2)
		defer c.End()
		for c.Loop() {
			got = d.Error()
		}
		assert.Equal(b, got, "tree: the body diverged at choice 12, where it requested other bounds", "the text")
	})
}
