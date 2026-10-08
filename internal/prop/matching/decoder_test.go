// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matching_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/alphabet"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// lineTerminators is the number of characters that . leaves out: \n, \r,
// U+0085, U+2028 and U+2029.
const lineTerminators = 5

// TestDecoder checks what the pieces of a pattern decode from stated
// choices: a literal, a sequence, an alternation, a repetition and a class,
// with the spans they open and the order of a class's members.
func TestDecoder(t *testing.T) {
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
			{name: "decodes index 0 of a class as its simplest member", text: `[A0a]`, give: []uint64{0}, want: "0"},
			{
				name: "decodes index 1 of a class as a lowercase letter after the digit",
				text: `[A0a]`,
				give: []uint64{1},
				want: "a",
			},
			{
				name: "decodes index 2 of a class as an uppercase letter last",
				text: `[A0a]`,
				give: []uint64{2},
				want: "A",
			},
			{name: "decodes the last index of a range as its last member", text: `[a-c]`, give: []uint64{2}, want: "c"},
			{name: "decodes an index of a negated class past its members", text: `[^a]`, give: []uint64{10}, want: "b"},
			{name: "decodes the index of a dot past the newline", text: `.`, give: []uint64{105}, want: "\v"},
			{
				name: "decodes the last index of the dot",
				text: `.`,
				give: []uint64{alphabet.Size - lineTerminators - 1},
				want: "\U0010FFFF",
			},
			{name: "decodes the last index of the digits", text: `\d`, give: []uint64{9}, want: "9"},
			{name: "decodes the last index of the word characters", text: `\w`, give: []uint64{62}, want: "_"},
			{name: "decodes index 0 of the spaces as a space", text: `\s`, give: []uint64{0}, want: " "},
			{name: "decodes index 1 of the spaces as a tab", text: `\s`, give: []uint64{1}, want: "\t"},
			{name: "decodes index 2 of the spaces as a newline", text: `\s`, give: []uint64{2}, want: "\n"},
			{name: "decodes index 3 of the spaces as a form feed", text: `\s`, give: []uint64{3}, want: "\f"},
			{
				name: "decodes the last index of the spaces as a carriage return",
				text: `\s`,
				give: []uint64{4},
				want: "\r",
			},
			{
				name: "decodes the target past the last index of the dot",
				text: `.`,
				give: []uint64{alphabet.Size - lineTerminators},
				want: "0",
			},
			{name: "decodes the target past the last index of the digits", text: `\d`, give: []uint64{10}, want: "0"},
			{
				name: "decodes the target past the last index of the word characters",
				text: `\w`,
				give: []uint64{63},
				want: "0",
			},
			{name: "decodes the target past the last index of the spaces", text: `\s`, give: []uint64{5}, want: " "},
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

// labels returns the labels of spans, in order.
func labels(spans []engine.Span) []string {
	out := make([]string, len(spans))
	for i, span := range spans {
		out[i] = span.Label
	}
	return out
}
