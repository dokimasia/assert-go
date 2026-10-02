// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pattern

import (
	"strings"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The labels of the spans that a pattern's pieces open.
const (
	// alternationLabel labels the span of an alternation's index and its
	// branch.
	alternationLabel = "alternation"
	// repeatLabel labels the span of a quantified piece's repetitions.
	repeatLabel = "repeat"
)

// node is a parsed piece of a pattern, which decodes its characters from a
// case.
type node interface {
	// emit makes the piece's choices on c and writes its characters to b.
	emit(c *engine.Case, b *strings.Builder)
}

// literal is one character, which makes no choice.
type literal rune

var _ node = literal(0)

// emit writes the character.
func (l literal) emit(_ *engine.Case, b *strings.Builder) {
	b.WriteRune(rune(l))
}

// sequence is pieces that decode one after another. An empty sequence
// decodes the empty string.
type sequence []node

var _ node = sequence(nil)

// emit decodes each piece in order.
func (s sequence) emit(c *engine.Case, b *strings.Builder) {
	for _, n := range s {
		n.emit(c, b)
	}
}

// alternation is two or more branches, of which a case decodes one.
type alternation struct {
	// branches are the branches, in their stated order.
	branches []node
	// bounds are the bounds of a branch's index, [0, len(branches) - 1].
	bounds choice.IntegerBounds
}

var _ node = alternation{}

// emit decodes the branch at an index that decides structure, in a span
// labelled alternation. The index's edge and its target are 0, the first
// branch.
func (a alternation) emit(c *engine.Case, b *strings.Builder) {
	c.Span(alternationLabel, func() {
		a.branches[c.Structure(a.bounds, 0).Magnitude()].emit(c, b)
	})
}

// repeat is a quantified piece.
type repeat struct {
	// item is the piece that repeats.
	item node
	// sizes are the numbers of repetitions that the quantifier allows.
	sizes choice.Sizes
}

var _ node = repeat{}

// emit decodes the repetitions as a collection, in a span labelled repeat.
// Its target is the fewest repetitions.
func (r repeat) emit(c *engine.Case, b *strings.Builder) {
	c.Span(repeatLabel, func() {
		engine.Collect(c, r.sizes, func() { r.item.emit(c, b) })
	})
}
