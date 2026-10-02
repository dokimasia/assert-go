// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pattern_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestNode checks what the pieces of a pattern decode from stated choices:
// a literal, a sequence, an alternation and a repetition, with the spans
// they open.
func TestNode(t *testing.T) {
	t.Parallel()

	t.Run("StringMatching", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			text string
			give []uint64
			want string
		}{
			{
				name: "decodes the branch that an alternation's index chooses",
				text: `(foo|bar)`,
				give: []uint64{1},
				want: "bar",
			},
			{name: "decodes a repetition for each continue flag", text: `a*`, give: []uint64{1, 1, 0}, want: "aa"},
			{
				name: "decodes a counted repetition whatever its forced flags record",
				text: `(ab|c){2}`,
				give: []uint64{0, 1, 0, 0, 1},
				want: "cab",
			},
			{name: "decodes nine repetitions for a count of nine", text: `a{9}`, want: "aaaaaaaaa"},
			{name: "decodes an empty branch as the empty string", text: `(a|)b`, give: []uint64{1}, want: "b"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, _ := decode(t, tt.text, tt.give...)
				assert.Equal(t, got, tt.want, "the decoded string")
			})
		}

		t.Run("decodes a sequence of literals without a choice", func(t *testing.T) {
			t.Parallel()
			got, e := decode(t, `abc`)
			assert.Equal(t, got, "abc", "the literals")
			assert.Empty(t, e.Case.Choices(), "no choice")
		})

		t.Run("records a span for each alternation, repetition and element", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, `(a|b)[cd]*`, 1, 1, 1, 0)
			assert.Equal(t, e.Case.Spans(), []engine.Span{
				{Label: "string-matching", Start: 0, End: 4, Depth: 0, Parent: -1},
				{Label: "alternation", Start: 0, End: 1, Depth: 1, Parent: 0},
				{Label: "repeat", Start: 1, End: 4, Depth: 1, Parent: 0},
				{Label: "element", Start: 1, End: 3, Depth: 2, Parent: 2},
			}, "the spans the definition's reference records")
		})

		t.Run("records no alternation span for one branch", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, `(ab)`)
			assert.Equal(t, labels(e.Case.Spans()), []string{"string-matching"}, "the string's span alone")
		})
	})
}
