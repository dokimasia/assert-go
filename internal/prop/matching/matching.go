// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matching

import (
	"strings"

	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/pattern"
)

// id is the definition's id of string-matching, which labels the span of
// every string it decodes.
const id = "string-matching"

// StringMatching returns the generator of the strings that text matches in
// full, for text of the portable subset. Its simplest value takes the first
// branch of each alternation, the fewest repetitions and the simplest
// character of each class. It returns an error that wraps
// [pattern.ErrOutside] for text outside the subset, or that is not UTF-8.
func StringMatching(text string) (engine.Generator[string], error) {
	parsed, err := pattern.Parse(text)
	if err != nil {
		return engine.Generator[string]{}, err
	}
	root := decoderOf(parsed)
	return engine.NewGenerator(id, func(c *engine.Case) string {
		var b strings.Builder
		c.Span(id, func() { root.emit(c, &b) })
		return b.String()
	}), nil
}
