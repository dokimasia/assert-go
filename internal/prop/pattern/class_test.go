// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pattern_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/alphabet"
	"go.dokimi.dev/assert/internal/prop/pattern"
)

// lineTerminators is the number of characters that . leaves out: \n, \r,
// U+0085, U+2028 and U+2029.
const lineTerminators = 5

// TestClass checks the members of the classes that Parse returns: the sets
// that the subset names, a negated class, a range and the order of a
// class's members.
func TestClass(t *testing.T) {
	t.Parallel()

	t.Run("Parse", func(t *testing.T) {
		t.Parallel()

		counts := []struct {
			name string
			text string
			want uint64
		}{
			{
				name: "returns every character but the line terminators for the dot",
				text: `.`,
				want: alphabet.Size - lineTerminators,
			},
			{name: "returns the ten ASCII digits for \\d", text: `\d`, want: 10},
			{name: "returns the 63 ASCII word characters for \\w", text: `\w`, want: 63},
			{name: "returns the five ASCII spaces for \\s", text: `\s`, want: 5},
			{name: "returns every character but one for a negated class", text: `[^a]`, want: alphabet.Size - 1},
			{name: "returns each character of a range", text: `[a-c]`, want: 3},
			{name: "returns a shorthand inside a class with the other members", text: `[\d_]`, want: 11},
		}
		for _, tt := range counts {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, classOf(t, tt.text).Count, tt.want, "the number of members")
			})
		}

		t.Run("returns the members in the order of the default alphabet", func(t *testing.T) {
			t.Parallel()
			got := classOf(t, `[A0a]`).Members
			zero, _ := alphabet.Index('0')
			lower, _ := alphabet.Index('a')
			upper, _ := alphabet.Index('A')
			assert.Equal(t, got, []alphabet.Interval{
				{First: zero, Last: zero}, {First: lower, Last: lower}, {First: upper, Last: upper},
			}, "the digit, the lowercase letter, the uppercase letter")
		})
	})
}

// classOf returns the class that text parses to, failing the test when it
// parses to another piece.
func classOf(t *testing.T, text string) pattern.Class {
	t.Helper()
	parsed, err := pattern.Parse(text)
	assert.NoError(t, err, "the pattern is in the portable subset")
	cl, ok := parsed.(pattern.Class)
	assert.True(t, ok, text+" parses to a class")
	return cl
}
