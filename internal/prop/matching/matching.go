// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matching

import (
	"strings"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/pattern"
)

// id is the definition's id of string-matching, which labels the span of
// every string it decodes.
const id = "string-matching"

// StringMatching returns the generator of the strings that text matches in
// full, for text of the portable subset. Its simplest value takes the first
// branch of each alternation, the fewest repetitions and the simplest
// character of each class. It runs backwards from a string that text
// matches in full, through the match that a backtracking engine finds
// first.
//
// # Errors
//
// StringMatching returns the fault of [pattern.Parse], of the kind
// [pattern.ErrOutside], for text outside the subset or that is not UTF-8.
// The generator's inverse returns a fault of the kind
// [engine.ErrCannotInvert] for a value that is no string of UTF-8, or that
// text does not match in full.
func StringMatching(text string) (engine.Generator[string], error) {
	parsed, err := pattern.Parse(text)
	if err != nil {
		return engine.Generator[string]{}, err
	}
	root := decoderOf(parsed)
	decode := func(c *engine.Case) string {
		var b strings.Builder
		c.Span(id, func() { root.emit(c, &b) })
		return b.String()
	}
	return engine.NewInvertible(id, decode, func(v any) ([]engine.Step, string, error) {
		s, ok := v.(string)
		if !ok || !utf8.ValidString(s) {
			return nil, "", fault.Of(engine.ErrCannotInvert, "%v is no string of UTF-8", v)
		}
		runes := []rune(s)
		for end, steps := range root.match(runes, 0) {
			if end == len(runes) {
				return steps, s, nil
			}
		}
		return nil, "", fault.Of(engine.ErrCannotInvert, "%s does not match %q in full", text, s)
	}), nil
}
