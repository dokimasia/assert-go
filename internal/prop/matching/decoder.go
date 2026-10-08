// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matching

import (
	"iter"
	"slices"
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

// decoder is a piece of a pattern that decodes its characters from a case,
// and runs backwards from them.
type decoder interface {
	// emit makes the piece's choices on c and writes its characters to b.
	emit(c *engine.Case, b *strings.Builder)
	// match yields each way the piece matches text from at, in the order
	// that a backtracking engine tries them: where the match ends, and the
	// steps of the choices that emit makes for it.
	match(text []rune, at int) iter.Seq2[int, []engine.Step]
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

// match yields the end of the character when text has it at at.
func (l literal) match(text []rune, at int) iter.Seq2[int, []engine.Step] {
	return func(yield func(int, []engine.Step) bool) {
		if at < len(text) && text[at] == rune(l) {
			yield(at+1, nil)
		}
	}
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

// match yields each way the pieces match text one after another from at.
func (s sequence) match(text []rune, at int) iter.Seq2[int, []engine.Step] {
	return func(yield func(int, []engine.Step) bool) {
		s.matchFrom(text, at, nil, yield)
	}
}

// matchFrom yields each way the pieces of s match text one after another
// from at, each after the steps that came before. It reports false once
// yield does.
func (s sequence) matchFrom(text []rune, at int, before []engine.Step, yield func(int, []engine.Step) bool) bool {
	if len(s) == 0 {
		return yield(at, before)
	}
	for middle, first := range s[0].match(text, at) {
		if !s[1:].matchFrom(text, middle, slices.Concat(before, first), yield) {
			return false
		}
	}
	return true
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

// match yields each way each branch matches text from at, the branches in
// their stated order, after the step of the branch's index.
func (a alternation) match(text []rune, at int) iter.Seq2[int, []engine.Step] {
	return func(yield func(int, []engine.Step) bool) {
		for i, branch := range a.branches {
			index := stepOf(a.bounds, uint64(i))
			for end, steps := range branch.match(text, at) {
				if !yield(end, slices.Concat([]engine.Step{index}, steps)) {
					return
				}
			}
		}
	}
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

// match yields each way the repetitions match text from at, the most
// repetitions first, each repetition after its continue flag and the last
// one before the stop flag.
func (r repeat) match(text []rune, at int) iter.Seq2[int, []engine.Step] {
	return func(yield func(int, []engine.Step) bool) {
		r.matchFrom(text, at, 0, yield)
	}
}

// matchFrom yields each way the repetitions from count on match text from
// at, the most first. A repetition that matches nothing is not repeated
// beyond the minimum, so the search ends. It reports false once yield does.
func (r repeat) matchFrom(text []rune, at, count int, yield func(int, []engine.Step) bool) bool {
	flag := r.sizes.FlagBounds(count)
	if flag.Hi() == choice.UintOf(1) {
		more := stepOf(flag, 1)
		for middle, item := range r.item.match(text, at) {
			if middle == at && count >= r.sizes.Min() {
				continue
			}
			kept := r.matchFrom(text, middle, count+1, func(end int, rest []engine.Step) bool {
				return yield(end, slices.Concat([]engine.Step{more}, item, rest))
			})
			if !kept {
				return false
			}
		}
	}
	if flag.Lo() == (choice.Int{}) {
		return yield(at, []engine.Step{stepOf(flag, 0)})
	}
	return true
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

// match yields the end of the character at at, after the step of its index
// among the members, when the class has it.
func (cl class) match(text []rune, at int) iter.Seq2[int, []engine.Step] {
	return func(yield func(int, []engine.Step) bool) {
		if at == len(text) {
			return
		}
		if offset, ok := cl.offset(text[at]); ok {
			yield(at+1, []engine.Step{stepOf(cl.bounds, offset)})
		}
	}
}

// offset returns the index of r among the members, and false for a
// character that is no member.
func (cl class) offset(r rune) (uint64, bool) {
	position, _ := alphabet.Index(r)
	offset := uint64(0)
	for _, v := range cl.members {
		if v.First <= position && position <= v.Last {
			return offset + uint64(position-v.First), true
		}
		offset += uint64(v.Last-v.First) + 1
	}
	return 0, false
}

// stepOf returns the step of the integer choice i under b, which admits it.
func stepOf(b choice.IntegerBounds, i uint64) engine.Step {
	return engine.Step{
		Bounds: choice.OfInteger(b),
		Value:  choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(i)},
	}
}
