// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pattern

import "go.dokimi.dev/assert/internal/prop/choice"

// Node is a parsed piece of a pattern: a [Literal], a [Sequence], an
// [Alternation], a [Repeat] or a [Class].
type Node interface {
	// piece marks the five kinds of piece, so no other type is a Node.
	piece()
}

// Literal is one character.
type Literal rune

// Sequence is pieces one after another. An empty sequence is the empty
// string.
type Sequence []Node

// Alternation is two or more branches, in their stated order.
type Alternation []Node

// Repeat is a quantified piece.
type Repeat struct {
	// Item is the piece that repeats.
	Item Node
	// Sizes are the numbers of repetitions that the quantifier allows.
	Sizes choice.Sizes
}

var (
	_ Node = Literal(0)
	_ Node = Sequence(nil)
	_ Node = Alternation(nil)
	_ Node = Repeat{}
	_ Node = Class{}
)

func (Literal) piece()     {}
func (Sequence) piece()    {}
func (Alternation) piece() {}
func (Repeat) piece()      {}
func (Class) piece()       {}
