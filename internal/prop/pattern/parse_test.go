// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pattern_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/pattern"
)

// outside is the text that every error of Parse starts with.
const outside = "pattern: outside the portable subset: "

// accepted are patterns that together contain every construct of the
// portable subset, from the definition's executable reference, with a
// count that states the digit 9 and a range of one character.
var accepted = []string{
	``,
	`abc`,
	`^abc$`,
	`^$`,
	`a|b|c`,
	`(foo|bar)baz`,
	`(?:ab)+`,
	`[a-z]+@[a-z]+\.com`,
	`\d{3}-\d{4}`,
	`\w+\s\w*`,
	`.`,
	`.{2,5}`,
	`[^a-z]`,
	`[-a]`,
	`[a-]`,
	`[a-z-]`,
	`[\]\[\\\-]`,
	`[\d_]x?`,
	`a{0}`,
	`a{2,}`,
	`(a|)+`,
	`x*y+z?`,
	`[à-ÿ]`,
	`[.$^*+?(){}|]`,
	`\.\*\+\?\(\)\[\]\{\}\|\^\$\\`,
	`a{1000}`,
	`[a-z]{9}`,
	`[x-x]`,
}

// atTheLimits are patterns whose nested counts multiply to the limit, or
// whose groups nest to it, with the open quantifiers that count as 1.
var atTheLimits = []string{
	`(?:a{10}){100}`,
	`(a{1000})*`,
	`(a{1000}){0,1}`,
	strings.Repeat("(", 100) + "a" + strings.Repeat(")", 100),
}

// tooDeep is a pattern whose groups nest one deeper than the limit.
var tooDeep = strings.Repeat("(", 101) + "a" + strings.Repeat(")", 101)

// maxFuzzedPattern is the longest pattern whose acceptance by RE2
// FuzzParse checks. The densest pattern of the subset this long, a{0,1000}
// repeated, compiles to 910,002 instructions, below RE2's limit of
// 3,355,443. A longer pattern can pass the limit.
const maxFuzzedPattern = 4096

// FuzzParse checks that Parse returns an error, and does not panic, for
// any text that it refuses, and that RE2 compiles every pattern of up to
// maxFuzzedPattern bytes that Parse accepts. The second check makes the
// subset portable to Go.
func FuzzParse(f *testing.F) {
	for _, text := range accepted {
		f.Add(text)
	}
	for _, text := range atTheLimits {
		f.Add(text)
	}
	f.Add(`(a{1000}){2}`)
	f.Add(tooDeep)
	f.Fuzz(func(t *testing.T, text string) {
		if _, err := pattern.Parse(text); err != nil || len(text) > maxFuzzedPattern {
			return
		}
		if _, err := regexp.Compile(text); err != nil {
			t.Fatalf("Parse accepts %q, which RE2 refuses: %v", text, err)
		}
	})
}

// TestParse checks that Parse accepts every construct of the portable
// subset, and refuses each construct outside it with an error that states
// the position and the fault.
func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("Parse", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the pieces of every construct of the subset", func(t *testing.T) {
			t.Parallel()
			for _, text := range accepted {
				_, err := pattern.Parse(text)
				assert.NoError(t, err, text)
			}
		})

		t.Run("returns the pieces of a pattern at the limits of the subset", func(t *testing.T) {
			t.Parallel()
			for _, text := range atTheLimits {
				_, err := pattern.Parse(text)
				assert.NoError(t, err, text)
			}
		})

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns ErrOutside for a quantifier after a quantifier",
				give: `a**`,
				want: `"a**" at 3: '*' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a lazy quantifier",
				give: `a+?`,
				want: `"a+?" at 3: '?' must be escaped here`,
			},
			{
				name: "returns ErrOutside for two counts",
				give: `a{2}{3}`,
				want: `"a{2}{3}" at 5: '{' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a quantifier with nothing to repeat",
				give: `*a`,
				want: `"*a" at 1: '*' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a quantifier with nothing to repeat in a later branch",
				give: `a|*`,
				want: `"a|*" at 3: '*' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a count that runs backwards",
				give: `a{2,1}`,
				want: `"a{2,1}" at 6: the count {2,1} runs backwards`,
			},
			{
				name: "returns ErrOutside for a count above the limit",
				give: `a{1001}`,
				want: `"a{1001}" at 6: the count 1001 is above 1000`,
			},
			{
				name: "returns ErrOutside for nested counts that multiply past the limit",
				give: `(a{1000}){2}`,
				want: `"(a{1000}){2}" at 12: nested counts multiply to 2000, above 1000`,
			},
			{
				name: "returns ErrOutside for three nested counts that multiply past the limit",
				give: `((a{100}){10}){2}`,
				want: `"((a{100}){10}){2}" at 17: nested counts multiply to 2000, above 1000`,
			},
			{
				name: "returns ErrOutside for an open count whose lower count multiplies past the limit",
				give: `(a{500}){3,}`,
				want: `"(a{500}){3,}" at 12: nested counts multiply to 1500, above 1000`,
			},
			{
				name: "returns ErrOutside for a count of 0, which counts as 1 in a product past the limit",
				give: `((a{1000}){0}){2}`,
				want: `"((a{1000}){0}){2}" at 17: nested counts multiply to 2000, above 1000`,
			},
			{
				name: "returns ErrOutside for groups nested past the limit",
				give: tooDeep,
				want: strconv.Quote(tooDeep) + " at 101: groups nest deeper than 100",
			},
			{
				name: "returns ErrOutside for a count with a leading zero",
				give: `a{01}`,
				want: `"a{01}" at 4: the count 01 has a leading zero`,
			},
			{
				name: "returns ErrOutside for a maximum with a leading zero",
				give: `a{1,02}`,
				want: `"a{1,02}" at 6: the count 02 has a leading zero`,
			},
			{
				name: "returns ErrOutside for a count without a minimum",
				give: `a{,3}`,
				want: `"a{,3}" at 2: a count has no digits`,
			},
			{
				name: "returns ErrOutside for a count without digits",
				give: `a{x}`,
				want: `"a{x}" at 2: a count has no digits`,
			},
			{
				name: "returns ErrOutside for a count with a digit outside ASCII",
				give: "a{\U00000661}",
				want: "\"a{\U00000661}\" at 2: a count has no digits",
			},
			{name: "returns ErrOutside for an unclosed count", give: `a{1`, want: `"a{1" at 3: a count is not closed`},
			{
				name: "returns ErrOutside for a count that a brace does not close",
				give: `a{1x}`,
				want: `"a{1x}" at 4: a count is not closed by }`,
			},
			{
				name: "returns ErrOutside for a lookahead",
				give: `(?=a)`,
				want: `"(?=a)" at 1: only the (?: group is in the portable subset`,
			},
			{
				name: "returns ErrOutside for a named group",
				give: `(?P<n>a)`,
				want: `"(?P<n>a)" at 1: only the (?: group is in the portable subset`,
			},
			{
				name: "returns ErrOutside for a word boundary",
				give: `\b`,
				want: `"\\b" at 2: \b is not in the portable subset`,
			},
			{
				name: "returns ErrOutside for a back reference",
				give: `\1`,
				want: `"\\1" at 2: \1 is not in the portable subset`,
			},
			{
				name: "returns ErrOutside for an escaped control character",
				give: `\n`,
				want: `"\\n" at 2: \n is not in the portable subset`,
			},
			{
				name: "returns ErrOutside for a negated shorthand",
				give: `\D`,
				want: `"\\D" at 2: \D is not in the portable subset`,
			},
			{
				name: "returns ErrOutside for a trailing backslash",
				give: `a\`,
				want: `"a\\" at 2: the pattern ends with a backslash`,
			},
			{
				name: "returns ErrOutside for an end anchor inside",
				give: `a$b`,
				want: `"a$b" at 2: '$' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a start anchor inside",
				give: `a^`,
				want: `"a^" at 2: '^' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a start anchor in a group",
				give: `(^a)`,
				want: `"(^a)" at 2: '^' must be escaped here`,
			},
			{
				name: "returns ErrOutside for an end anchor in a group",
				give: `(a$)`,
				want: `"(a$)" at 3: '$' must be escaped here`,
			},
			{
				name: "returns ErrOutside for an unclosed group that ends with an anchor",
				give: `(a$`,
				want: `"(a$" at 3: a group is not closed by )`,
			},
			{name: "returns ErrOutside for an unclosed group", give: `(a`, want: `"(a" at 2: a group is not closed`},
			{name: "returns ErrOutside for an unopened group", give: `)`, want: `")" at 0: ')' is not expected`},
			{
				name: "returns ErrOutside for an intersection",
				give: `[a&&b]`,
				want: `"[a&&b]" at 4: "&&" is reserved inside a class`,
			},
			{
				name: "returns ErrOutside for a range that ends with a reserved pair",
				give: `[a--]`,
				want: `"[a--]" at 4: "--" is reserved inside a class`,
			},
			{
				name: "returns ErrOutside for a range that starts with a reserved pair",
				give: `[--a]`,
				want: `"[--a]" at 2: "--" is reserved inside a class`,
			},
			{
				name: "returns ErrOutside for a nested class",
				give: `[[a]]`,
				want: `"[[a]]" at 2: [ must be escaped inside a class`,
			},
			{
				name: "returns ErrOutside for a range to a shorthand",
				give: `[a-\d]`,
				want: `"[a-\\d]" at 5: \d is not in the portable subset`,
			},
			{
				name: "returns ErrOutside for a range from a shorthand",
				give: `[\d-z]`,
				want: `"[\\d-z]" at 4: a hyphen inside a class must be escaped`,
			},
			{
				name: "returns ErrOutside for a range that runs backwards",
				give: `[z-ab]`,
				want: `"[z-ab]" at 4: the range z-a runs backwards`,
			},
			{
				name: "returns ErrOutside for a hyphen inside a class",
				give: `[a-z-0]`,
				want: `"[a-z-0]" at 5: a hyphen inside a class must be escaped`,
			},
			{
				name: "returns ErrOutside for an escape inside a class",
				give: `[\n]`,
				want: `"[\\n]" at 3: \n is not in the portable subset`,
			},
			{
				name: "returns ErrOutside for a class that ends inside an escape",
				give: `[a\`,
				want: `"[a\\" at 3: a class ends inside an escape`,
			},
			{name: "returns ErrOutside for an empty class", give: `[]`, want: `"[]" at 2: a class is empty`},
			{name: "returns ErrOutside for an empty negated class", give: `[^]`, want: `"[^]" at 3: a class is empty`},
			{name: "returns ErrOutside for an unclosed class", give: `[a`, want: `"[a" at 2: a class is not closed`},
			{
				name: "returns ErrOutside for a negated class without a member",
				give: "[^\x00-\U0010FFFF]",
				want: `"[^\x00-\U0010ffff]" at 6: a class has no member`,
			},
			{
				name: "returns ErrOutside for a bare closing bracket",
				give: `]`,
				want: `"]" at 1: ']' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a bare closing brace",
				give: `}`,
				want: `"}" at 1: '}' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a bare opening brace",
				give: `{`,
				want: `"{" at 1: '{' must be escaped here`,
			},
			{
				name: "returns ErrOutside for a pattern that is not UTF-8",
				give: "\xed\xa0\x80",
				want: `"\xed\xa0\x80" is not UTF-8`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := pattern.Parse(tt.give)
				assert.ErrorIs(t, err, pattern.ErrOutside, "the pattern is outside the subset")
				assert.Equal(t, err.Error(), outside+tt.want, "the position and the fault")
			})
		}
	})
}
