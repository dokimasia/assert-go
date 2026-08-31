// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

// Defined types over the kinds each family reads. A surface that
// switches on concrete types alone passes the plain cases and fails
// these, which is why every table carries one.
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
		{Name: "bytes answer as text", Args: []any{[]byte("store: x"), "store: "}},
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
			Name:      "a non-text value reports rather than panicking",
			Args:      []any{42, "4"},
			Fails:     true,
			Assertion: "has-prefix",
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
			Name:      "a non-text value reports rather than panicking",
			Args:      []any{42, "2"},
			Fails:     true,
			Assertion: "has-suffix",
		},
	}
}

// MatchesCases are the cases every surface's pattern assertion must
// produce. Drive them with [RunPair].
//
// A pattern that does not compile is a failure rather than a panic: a
// test with a broken pattern has established nothing, and should say
// so where every other failure is reported.
func MatchesCases() []Case {
	return []Case{
		{Name: "an anchored pattern matches the whole value", Args: []any{"deadbeef", `^[0-9a-f]+$`}},
		{Name: "an unanchored pattern matches anywhere", Args: []any{"id=deadbeef;", `[0-9a-f]{8}`}},
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
			Name:      "a pattern that does not compile reports rather than panicking",
			Args:      []any{"anything", `([unclosed`},
			Fails:     true,
			Assertion: "matches",
		},
		{
			Name:      "a non-text value reports rather than panicking",
			Args:      []any{42, `\d`},
			Fails:     true,
			Assertion: "matches",
		},
	}
}
