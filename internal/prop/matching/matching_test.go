// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matching_test

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/matching"
	"go.dokimi.dev/assert/internal/prop/pattern"
	"go.dokimi.dev/assert/internal/prop/token"
)

// The pins of a test body.
const (
	// drawn is the label of the one string a test body draws.
	drawn = "s"
	// seeds is the number of seeds whose strings the full-match check
	// decodes for each pattern.
	seeds = 200
	// referenceSeed is the seed of every run whose counterexample the tests
	// pin from the definition's executable reference.
	referenceSeed = 7
)

// The allocations of string-matching, measured.
const (
	// stringMatchingAllocs are the allocations of StringMatching on the
	// pattern of a hexadecimal identifier: the parser and its characters,
	// the parsed pieces and classes, the decoders built from them, and the
	// generator's decodes.
	stringMatchingAllocs = 24
	// drawAllocs are the allocations of a whole replayed case that draws a
	// string of that pattern from no choices.
	drawAllocs = 25
)

// identifier is the pattern of the benchmarks and the allocation checks.
const identifier = `[a-f0-9]{4}-\d{2}`

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

// pinned are the strings and the choices of the first cases of a seed, from
// the definition's executable reference, for patterns that together use
// every construct.
var pinned = []struct {
	name    string
	text    string
	seed    uint64
	values  []string
	choices []string
}{
	{
		name:   "returns the pinned strings of classes and counted repetitions",
		text:   identifier,
		seed:   42,
		values: []string{"35de-85", "51c2-81", "d93f-13", "0e51-34", "8c18-48", "2644-04"},
		choices: []string{
			"prop1:AAEAAwABAAUAAQANAAEADgAAAAEACAABAAUAAA",
			"prop1:AAEABQABAAEAAQAMAAEAAgAAAAEACAABAAEAAA",
			"prop1:AAEADQABAAkAAQADAAEADwAAAAEAAQABAAMAAA",
			"prop1:AAEAAAABAA4AAQAFAAEAAQAAAAEAAwABAAQAAA",
			"prop1:AAEACAABAAwAAQABAAEACAAAAAEABAABAAgAAA",
			"prop1:AAEAAgABAAYAAQAEAAEABAAAAAEAAAABAAQAAA",
		},
	},
	{
		name: "returns the pinned strings of alternations and unbounded repetitions",
		text: `(foo|ba[rz])+\.?`,
		seed: 7,
		values: []string{
			"bazfoobazfoobarfoofoobarbazfoo",
			"foobazbarbazfoobazbarfoobazfoofoobazbazbarbazfoo.",
			"bar",
			"baz",
			"foofoofoofoobazbazfoobarfoobar.",
			"bazbarbar",
		},
		choices: []string{
			"prop1:AAEAAQABAAEAAAABAAEAAQABAAAAAQABAAAAAQAAAAEAAAABAAEAAAABAAEAAQABAAAAAAAA",
			"prop1:AAEAAAABAAEAAQABAAEAAAABAAEAAQABAAAAAQABAAEAAQABAAAAAQAAAAEAAQABAAEAAAABAAAAAQ" +
				"ABAAEAAQABAAEAAQABAAAAAQABAAEAAQAAAAAAAQAA",
			"prop1:AAEAAQAAAAAAAA",
			"prop1:AAEAAQABAAAAAA",
			"prop1:AAEAAAABAAAAAQAAAAEAAAABAAEAAQABAAEAAQABAAAAAQABAAAAAQAAAAEAAQAAAAAAAQAA",
			"prop1:AAEAAQABAAEAAQAAAAEAAQAAAAAAAA",
		},
	},
	{
		name:   "returns the pinned strings of the dot",
		text:   `.{2,5}`,
		seed:   3,
		values: []string{"\u001f\U0000121a1", "0c\U0000a16b1", "\U0000fb2a\U00066322", "\u0080\U000e56a95\U0000f5eb"},
		choices: []string{
			"prop1:AAEAfAABAJckAAEAAQAA",
			"prop1:AAEAAAABAAwAAQDmwgIAAQABAAA",
			"prop1:AAEApeYDAAEAnbYZAAA",
			"prop1:AAEAfgABAKSdOQABAAUAAQDm2wMAAA",
		},
	},
	{
		name:    "returns the pinned strings of a negated class and the shorthands",
		text:    `[^a-z]\w\s`,
		seed:    1,
		values:  []string{"\U00008f20e ", "\U0000419e2\t", "\u00964 ", "0c "},
		choices: []string{"prop1:AIaeAgAOAAA", "prop1:AISDAQACAAE", "prop1:AHwABAAA", "prop1:AAAADAAA"},
	},
	{
		name:   "returns the pinned strings of a group with an empty branch and an open count",
		text:   `(?:ab|c|)*x{2,}`,
		seed:   5,
		values: []string{"xxxxx", "cabababccxxxxxxxx", "cccabxxxxxxxxx", "ccccccabababababxx"},
		choices: []string{
			"prop1:AAAAAQABAAEAAQABAAA",
			"prop1:AAEAAQABAAIAAQAAAAEAAgABAAAAAQACAAEAAAABAAEAAQABAAEAAgABAAIAAQACAAAAAQABAAEAAQABAAEAAQABAAA",
			"prop1:AAEAAgABAAEAAQABAAEAAQABAAIAAQAAAAAAAQABAAEAAQABAAEAAQABAAEAAA",
			"prop1:AAEAAQABAAEAAQABAAEAAQABAAIAAQACAAEAAQABAAEAAQACAAEAAAABAAAAAQAAAAEAAgABAAAAAQAAAAAAAQABAAA",
		},
	},
}

// TestMatching checks the strings that StringMatching decodes: the simplest
// match, the strings of a seed that the definition pins, a full match of
// every string, the counterexamples of runs that shrink them, and the
// refusal of a pattern outside the portable subset.
func TestMatching(t *testing.T) {
	t.Parallel()

	t.Run("StringMatching", func(t *testing.T) {
		t.Parallel()

		targets := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns the simplest character of each class from no choices",
				give: `[a-z]+@[a-z]+\.com`,
				want: "a@a.com",
			},
			{name: "returns the first branch from no choices", give: `(foo|bar)baz`, want: "foobaz"},
			{name: "returns the fewest repetitions from no choices", give: `x{3}`, want: "xxx"},
			{name: "returns the simplest member of each shorthand from no choices", give: `\d\w\s`, want: "00 "},
			{name: "returns the simplest character of the dot from no choices", give: `.`, want: "0"},
			{name: "returns the simplest character outside a negated class from no choices", give: `[^0-9]`, want: "a"},
			{name: "returns no repetition of an optional piece from no choices", give: `a?`, want: ""},
			{name: "returns the literals between the anchors from no choices", give: `^abc$`, want: "abc"},
			{name: "returns the simplest member of a class in any stated order", give: `[A0a]`, want: "0"},
			{name: "returns the simplest member of a class with a shorthand", give: `[\d_]`, want: "0"},
		}
		for _, tt := range targets {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, _ := decode(t, tt.give)
				assert.Equal(t, got, tt.want, "the simplest match")
			})
		}

		for _, tt := range pinned {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				g := generatorOf(t, tt.text)
				values, choices := make([]string, len(tt.values)), make([]string, len(tt.values))
				for i := range tt.values {
					body := func(c *engine.Case) { values[i] = engine.Draw(c, g, drawn) }
					choices[i] = token.Encode(engine.Generate(body, tt.seed, uint64(i), nil).Case.Choices())
				}
				assert.Equal(t, values, tt.values, "the strings of the first cases")
				assert.Equal(t, choices, tt.choices, "the choices of the first cases")
			})
		}

		t.Run("returns strings that the pattern matches in full", func(t *testing.T) {
			t.Parallel()
			for _, text := range accepted {
				g, whole := generatorOf(t, text), regexp.MustCompile(`^(?:`+text+`)$`)
				for seed := range uint64(seeds) {
					var got string
					engine.Generate(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, seed, 0, nil)
					assert.True(t, whole.MatchString(got), text+" matches "+got)
				}
			}
		})

		t.Run("returns the error of a pattern outside the portable subset", func(t *testing.T) {
			t.Parallel()
			_, err := matching.StringMatching(`a**`)
			assert.ErrorIs(t, err, pattern.ErrOutside, "the pattern is outside the subset")
		})

		shrinks := []struct {
			name  string
			text  string
			fails func(string) bool
			want  reference
		}{
			{
				name:  "returns the counterexample of a length that the definition pins",
				text:  `[a-z]+[0-9]`,
				fails: func(s string) bool { return len(s) >= 4 },
				want: reference{
					value:  "aaa0",
					token:  "prop1:AAEAAAABAAAAAQAAAAAAAA",
					runs:   31,
					calls:  34,
					digest: "4c66d19f709c5e3c23860262adf1ec10a6621523ff2754bd505fec0f88c594b9",
				},
			},
			{
				name:  "returns the counterexample of a branch chosen twice that the definition pins",
				text:  `(cat|dog)+`,
				fails: func(s string) bool { return strings.Count(s, "dog") >= 2 },
				want: reference{
					value:  "dogdog",
					token:  "prop1:AAEAAQABAAEAAA",
					runs:   19,
					calls:  22,
					digest: "0010fa6bbaafd80e2fa0f62cb8e8c143e6653bb83484bfcf474337421bef9c71",
				},
			},
			{
				name:  "returns the counterexample of a class member that the definition pins",
				text:  `[A0a]{3}`,
				fails: func(s string) bool { return strings.Contains(s, "A") },
				want: reference{
					value:  "00A",
					token:  "prop1:AAEAAAABAAAAAQACAAA",
					runs:   14,
					calls:  17,
					digest: "3ba9bbd1038ce768cfa32a889bcd45a6bbd2edaf23d687c857c1ef7d12efe2ba",
				},
			},
		}
		for _, tt := range shrinks {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, trace := traced(generatorOf(t, tt.text), tt.fails)
				assert.Equal(t, got.Outcome, engine.Counterexample, "a counterexample")
				assert.Equal(t, got.Explanation, []engine.Explained{
					{Label: drawn, Value: tt.want.value, Relevance: engine.ValueMatters},
				}, "the minimal string, whose value matters")
				assert.Equal(t, got.Token, tt.want.token, "the minimal case's token")
				assert.Equal(t, got.Runs, tt.want.runs, "the runs that shrinking and explaining spent")
				assert.Equal(t, len(trace), tt.want.calls, "the calls of the body")
				assert.Equal(t, digestOf(trace), tt.want.digest, "the choices of every call, in order")
			})
		}

		t.Run("returns the passing run that the definition pins", func(t *testing.T) {
			t.Parallel()
			got, trace := traced(generatorOf(t, `[a-z]+[0-9]`), func(string) bool { return false })
			assert.Equal(t, got.Outcome, engine.Passed, "no string fails")
			assert.Equal(t, got.Cases, engine.DefaultCases, "every case is valid")
			assert.Equal(t, len(trace), 104, "the calls of the body")
			assert.Equal(t, digestOf(trace), "d287ab6c19371ecd440152beda25b6f592f43d1a46ff73d847c6ec351df7ca3a",
				"the choices of every call, in order")
		})
	})
}

// TestMatchingZeroAlloc checks the allocation ceilings of StringMatching
// and of a whole replayed case that draws one of its strings.
func TestMatchingZeroAlloc(t *testing.T) {
	g := generatorOf(t, identifier)
	body := func(c *engine.Case) { engine.Draw(c, g, drawn) }
	assert.MaxAllocs(t, func() { _, _ = matching.StringMatching(identifier) }, stringMatchingAllocs,
		"StringMatching allocates the parsed pattern and its decoder")
	assert.MaxAllocs(t, func() { engine.Replay(body, nil, nil) }, drawAllocs, "a replayed draw allocates its case")
}

// BenchmarkMatching measures StringMatching, and a whole replayed case that
// draws one of its strings.
func BenchmarkMatching(b *testing.B) {
	b.Run("StringMatching", func(b *testing.B) {
		var got engine.Generator[string]
		c := bench.Start(b).MaxAllocs(stringMatchingAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = matching.StringMatching(identifier)
		}
		assert.Equal(b, got.ID(), "string-matching", "the id")
	})

	b.Run("Draw", func(b *testing.B) {
		var got string
		g := generatorOf(b, identifier)
		body := func(c *engine.Case) { got = engine.Draw(c, g, drawn) }
		c := bench.Start(b).MaxAllocs(drawAllocs)
		defer c.End()
		for c.Loop() {
			engine.Replay(body, nil, nil)
		}
		assert.Equal(b, got, "0000-00", "the simplest identifier")
	})
}

// reference is what the definition's executable reference reports for a
// run that shrinks: the minimal string, its token, the runs that shrinking
// and explaining spent, and the number of calls of the body with a SHA-256
// digest of the token of each call's choices, joined by newlines.
type reference struct {
	// value is the minimal string.
	value string
	// token is the minimal case's replay token.
	token string
	// runs are the runs that shrinking and explaining spent.
	runs int
	// calls is the number of calls of the body.
	calls int
	// digest is the digest of every call's choices.
	digest string
}

// generatorOf returns the generator of text, failing the test when text is
// outside the portable subset.
func generatorOf(tb testing.TB, text string) engine.Generator[string] {
	tb.Helper()
	g, err := matching.StringMatching(text)
	assert.NoError(tb, err, "the pattern is in the portable subset")
	return g
}

// decode returns the string that text decodes from a case replaying the
// integer choices of values, with the run of that case.
func decode(tb testing.TB, text string, values ...uint64) (string, engine.Execution) {
	tb.Helper()
	g := generatorOf(tb, text)
	choices := make([]choice.Choice, len(values))
	for i, v := range values {
		choices[i] = choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)}
	}
	var got string
	e := engine.Replay(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, choices, nil)
	return got, e
}

// traced runs a property of the reference seed that draws a string of g and
// fails when fails reports true for it, and returns the result and the
// token of the choices of each call of the body, in call order.
func traced(g engine.Generator[string], fails func(string) bool) (engine.Result, []string) {
	var trace []string
	result := engine.Run(func(c *engine.Case) {
		defer func() { trace = append(trace, token.Encode(c.Choices())) }()
		if fails(engine.Draw(c, g, drawn)) {
			c.Report(assert.Failure{Assertion: "fails"}, false)
		}
	}, engine.Settings{
		Seed:       referenceSeed,
		Cases:      engine.DefaultCases,
		MaxChoices: engine.MaxChoices,
		Shrink:     engine.DefaultShrink,
		Explain:    true,
	})
	return result, trace
}

// digestOf returns the SHA-256 digest of the trace joined by newlines, in
// hexadecimal.
func digestOf(trace []string) string {
	sum := sha256.Sum256([]byte(strings.Join(trace, "\n")))
	return hex.EncodeToString(sum[:])
}

// labels returns the labels of spans, in order.
func labels(spans []engine.Span) []string {
	out := make([]string, len(spans))
	for i, span := range spans {
		out[i] = span.Label
	}
	return out
}
