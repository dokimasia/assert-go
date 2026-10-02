// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pattern

import (
	"errors"
	"strings"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// id is the definition's id of string-matching, which labels the span of
// every string it decodes.
const id = "string-matching"

// ErrOutside reports a pattern outside the portable subset.
var ErrOutside = errors.New("pattern: outside the portable subset")

// StringMatching returns the generator of the strings that text matches in
// full, for text of the portable subset. Its simplest value takes the first
// branch of each alternation, the fewest repetitions and the simplest
// character of each class. It returns an error that wraps [ErrOutside] for
// text outside the subset, or that is not UTF-8.
func StringMatching(text string) (engine.Generator[string], error) {
	p, err := newParser(text)
	if err != nil {
		return engine.Generator[string]{}, err
	}
	root, err := p.pattern()
	if err != nil {
		return engine.Generator[string]{}, err
	}
	return engine.NewGenerator(id, func(c *engine.Case) string {
		var b strings.Builder
		c.Span(id, func() { root.emit(c, &b) })
		return b.String()
	}), nil
}
