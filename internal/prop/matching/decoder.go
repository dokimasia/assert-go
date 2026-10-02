// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matching

import (
	"strings"

	"go.dokimi.dev/assert/internal/prop/alphabet"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/pattern"
)

// The labels of the spans that a pattern's pieces open.
const (
	// alternationLabel labels the span of an alternation's index and its
	// branch.
	alternationLabel = "alternation"
	// repeatLabel labels the span of a quantified piece's repetitions.
	repeatLabel = "repeat"
)

// decoder is a piece of a pattern that decodes its characters from a case.
type decoder interface {
	// emit makes the piece's choices on c and writes its characters to b.
	emit(c *engine.Case, b *strings.Builder)
}

// decoderOf returns the decoder of a parsed piece. A piece that is none of
// the first four kinds is a class, the fifth and last kind that pattern
// states.
func decoderOf(n pattern.Node) decoder {
	switch n := n.(type) {
	case pattern.Literal:
		return literal(n)
	case pattern.Sequence:
		items := make(sequence, len(n))
		for i, item := range n {
			items[i] = decoderOf(item)
		}
		return items
	case pattern.Alternation:
		branches := make([]decoder, len(n))
		for i, branch := range n {
			branches[i] = decoderOf(branch)
		}
		bounds := choice.MustIntegerBounds(choice.Int{}, choice.UintOf(uint64(len(n)-1)))
		return alternation{branches: branches, bounds: bounds}
	case pattern.Repeat:
		return repeat{item: decoderOf(n.Item), sizes: n.Sizes}
	default:
		cl := n.(pattern.Class)
		return class{members: cl.Members, bounds: choice.MustIntegerBounds(choice.Int{}, choice.UintOf(cl.Count-1))}
	}
}

// literal is one character, which makes no choice.
type literal rune

// emit writes the character.
func (l literal) emit(_ *engine.Case, b *strings.Builder) {
	b.WriteRune(rune(l))
}

// sequence is pieces that decode one after another. An empty sequence
// decodes the empty string.
type sequence []decoder

// emit decodes each piece in order.
func (s sequence) emit(c *engine.Case, b *strings.Builder) {
	for _, n := range s {
		n.emit(c, b)
	}
}

// alternation is two or more branches, of which a case decodes one.
type alternation struct {
	// branches are the branches, in their stated order.
	branches []decoder
	// bounds are the bounds of a branch's index, [0, len(branches) - 1].
	bounds choice.IntegerBounds
}

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
	item decoder
	// sizes are the numbers of repetitions that the quantifier allows.
	sizes choice.Sizes
}

// emit decodes the repetitions as a collection, in a span labelled repeat.
// Its target is the fewest repetitions.
func (r repeat) emit(c *engine.Case, b *strings.Builder) {
	c.Span(repeatLabel, func() {
		engine.Collect(c, r.sizes, func() { r.item.emit(c, b) })
	})
}

// class is one character of a set, chosen by an index over the set's
// members in the order of the default alphabet. Index 0, the target, is
// the simplest member.
type class struct {
	// members are the members' indices in the default alphabet, sorted,
	// neither overlapping nor touching.
	members []alphabet.Interval
	// bounds are the bounds of a member's index, [0, members - 1].
	bounds choice.IntegerBounds
}

// emit writes the member at an index that the case chooses: a value choice
// that the random phase draws anew.
func (cl class) emit(c *engine.Case, b *strings.Builder) {
	offset := c.Integer(cl.bounds).Magnitude()
	for _, v := range cl.members {
		width := uint64(v.Last-v.First) + 1
		if offset < width {
			b.WriteRune(alphabet.Rune(v.First + uint32(offset)))
			return
		}
		offset -= width
	}
}
