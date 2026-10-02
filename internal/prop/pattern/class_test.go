// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pattern_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/alphabet"
)

// lineTerminators is the number of characters that . leaves out: \n, \r,
// U+0085, U+2028 and U+2029.
const lineTerminators = 5

// TestClass checks the character a class decodes for an index: the order
// of its members, the members of ., \d, \w and \s, and a negated class.
func TestClass(t *testing.T) {
	t.Parallel()

	t.Run("StringMatching", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			text  string
			index uint64
			want  string
		}{
			{name: "decodes index 0 of a class as its simplest member", text: `[A0a]`, index: 0, want: "0"},
			{
				name:  "decodes index 1 of a class as a lowercase letter after the digit",
				text:  `[A0a]`,
				index: 1,
				want:  "a",
			},
			{name: "decodes index 2 of a class as an uppercase letter last", text: `[A0a]`, index: 2, want: "A"},
			{name: "decodes the last index of a range as its last member", text: `[a-c]`, index: 2, want: "c"},
			{name: "decodes an index of a negated class past its members", text: `[^a]`, index: 10, want: "b"},
			{name: "decodes the index of a dot past the newline", text: `.`, index: 105, want: "\v"},
			{
				name:  "decodes the last index of the dot",
				text:  `.`,
				index: alphabet.Size - lineTerminators - 1,
				want:  "\U0010FFFF",
			},
			{name: "decodes the last index of the digits", text: `\d`, index: 9, want: "9"},
			{name: "decodes the last index of the word characters", text: `\w`, index: 62, want: "_"},
			{name: "decodes index 0 of the spaces as a space", text: `\s`, index: 0, want: " "},
			{name: "decodes index 1 of the spaces as a tab", text: `\s`, index: 1, want: "\t"},
			{name: "decodes index 2 of the spaces as a newline", text: `\s`, index: 2, want: "\n"},
			{name: "decodes index 3 of the spaces as a form feed", text: `\s`, index: 3, want: "\f"},
			{name: "decodes the last index of the spaces as a carriage return", text: `\s`, index: 4, want: "\r"},
			{
				name:  "decodes the target past the last index of the dot",
				text:  `.`,
				index: alphabet.Size - lineTerminators,
				want:  "0",
			},
			{name: "decodes the target past the last index of the digits", text: `\d`, index: 10, want: "0"},
			{name: "decodes the target past the last index of the word characters", text: `\w`, index: 63, want: "0"},
			{name: "decodes the target past the last index of the spaces", text: `\s`, index: 5, want: " "},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, _ := decode(t, tt.text, tt.index)
				assert.Equal(t, got, tt.want, "the member at the index")
			})
		}
	})
}
