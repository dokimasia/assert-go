// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

// Defined types over the kinds that each family reads. A surface that
// switches on concrete types alone passes the plain cases and fails
// these, so every table contains one.
type (
	name    string
	digits  []byte
	celsius float64
	ids     []int
)

// HasPrefixCases are the cases every surface's prefix assertion must
// produce. Drive them with [RunPair].
func HasPrefixCases() []Case {
	return []Case{
		{Name: "a matching prefix passes", Args: []any{"store: missing", "store: "}},
		{Name: "bytes read as text", Args: []any{[]byte("store: x"), "store: "}},
		{Name: "an empty prefix passes", Args: []any{"anything", ""}},
		{Name: "a defined string type reads as text", Args: []any{name("store: x"), "store: "}},
		{Name: "the whole string is a prefix of itself", Args: []any{"abc", "abc"}},
		{
			Name:      "a wrong prefix reports both strings",
			Args:      []any{"cache: missing", "store: "},
			Fails:     true,
			Assertion: "has-prefix",
		},
		{
			Name:      "a value that is not text reports",
			Args:      []any{42, "4"},
			Fails:     true,
			Assertion: "has-prefix",
			Detail:    map[string]any{"got": 42, "prefix": "4"},
		},
	}
}

// HasSuffixCases are the cases every surface's suffix assertion must
// produce. Drive them with [RunPair].
func HasSuffixCases() []Case {
	return []Case{
		{Name: "a matching suffix passes", Args: []any{"types.gen.go", ".gen.go"}},
		{Name: "an empty suffix passes", Args: []any{"anything", ""}},
		{Name: "a defined byte slice reads as text", Args: []any{digits("a.gen.go"), ".gen.go"}},
		{
			Name:      "a wrong suffix reports both strings",
			Args:      []any{"types.go", ".gen.go"},
			Fails:     true,
			Assertion: "has-suffix",
		},
		{
			Name:      "a value that is not text reports",
			Args:      []any{42, "2"},
			Fails:     true,
			Assertion: "has-suffix",
			Detail:    map[string]any{"got": 42, "suffix": "2"},
		},
	}
}

// MatchesCases are the cases every surface's pattern assertion must
// produce. Drive them with [RunPair].
//
// A pattern that does not compile, or that is outside the portable
// subset, fails the assertion and does not panic, because a test with
// such a pattern has established nothing.
func MatchesCases() []Case {
	return []Case{
		{Name: "an anchored pattern matches the whole value", Args: []any{"deadbeef", `^[0-9a-f]+$`}},
		{Name: "an unanchored pattern matches anywhere", Args: []any{"id=deadbeef;", `[0-9a-f]{8}`}},
		{Name: "a dot matches a character that ends no line", Args: []any{"é", `^.$`}},
		{Name: "a dot in a class matches a dot", Args: []any{".", `^[.]$`}},
		{Name: "a dot after another member of a class matches a dot", Args: []any{".", `^[a.]$`}},
		{Name: "an escaped dot matches a dot", Args: []any{".", `^\.$`}},
		{
			Name:      "a dot matches no carriage return",
			Args:      []any{"\r", `^.$`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a dot matches no line separator",
			Args:      []any{"\U00002028", `^.$`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a dot in a class matches no other character",
			Args:      []any{"a", `^[.]$`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "an escaped dot matches no other character",
			Args:      []any{"a", `^\.$`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a dot after an escaped character matches no carriage return",
			Args:      []any{".\r", `^\..$`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a dot after a class matches no carriage return",
			Args:      []any{"a\r", `^[a].$`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a flag group is outside the subset and reports",
			Args:      []any{"A", `(?i)a`},
			Fails:     true,
			Assertion: "matches",
			Detail:    map[string]any{"got": "A", "pattern": `(?i)a`},
		},
		{
			Name:      "a word boundary is outside the subset and reports",
			Args:      []any{"a b", `\bb`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a non-matching pattern reports both",
			Args:      []any{"zzz", `^[0-9a-f]+$`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "an anchored pattern rejects a partial match",
			Args:      []any{"id=deadbeef", `^[0-9a-f]+$`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a pattern that does not compile reports",
			Args:      []any{"anything", `([unclosed`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a value that is not text reports",
			Args:      []any{42, `\d`},
			Fails:     true,
			Assertion: "matches",
			Detail:    map[string]any{"got": 42, "pattern": `\d`},
		},
	}
}
