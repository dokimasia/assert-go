// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pattern_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/pattern"
)

// TestNode checks the pieces that Parse returns: a literal, a sequence, an
// alternation and a repetition, and the anchors and groups that leave no
// piece of their own.
func TestNode(t *testing.T) {
	t.Parallel()

	unbounded, err := choice.NewUnboundedSizes(0)
	assert.NoError(t, err, "the sizes of *")
	optional, err := choice.NewSizes(0, 1)
	assert.NoError(t, err, "the sizes of ?")

	t.Run("Parse", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want pattern.Node
		}{
			{name: "returns a literal for one character", give: `a`, want: pattern.Literal('a')},
			{name: "returns an empty sequence for the empty pattern", give: ``, want: pattern.Sequence(nil)},
			{
				name: "returns a sequence of the pieces in order",
				give: `ab`,
				want: pattern.Sequence{pattern.Literal('a'), pattern.Literal('b')},
			},
			{
				name: "returns an alternation of the branches in order",
				give: `a|bc`,
				want: pattern.Alternation{
					pattern.Literal('a'),
					pattern.Sequence{pattern.Literal('b'), pattern.Literal('c')},
				},
			},
			{
				name: "returns a repetition of a piece with the sizes of its quantifier",
				give: `a*`,
				want: pattern.Repeat{Item: pattern.Literal('a'), Sizes: unbounded},
			},
			{
				name: "returns the piece of a group, which adds none of its own",
				give: `(?:a)?`,
				want: pattern.Repeat{Item: pattern.Literal('a'), Sizes: optional},
			},
			{
				name: "returns the pieces between the anchors, which add none",
				give: `^ab$`,
				want: pattern.Sequence{pattern.Literal('a'), pattern.Literal('b')},
			},
			{name: "returns an escaped metacharacter as a literal", give: `\.`, want: pattern.Literal('.')},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := pattern.Parse(tt.give)
				assert.NoError(t, err, "the pattern is in the portable subset")
				assert.Equal(t, got, tt.want, "the pieces")
			})
		}
	})
}
