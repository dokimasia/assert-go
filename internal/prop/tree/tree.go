// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tree

import (
	"math/bits"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// NodeLimit is 2^20, the most nodes that the tree of a run contains.
const NodeLimit = 1048576

// root is the index of the root node.
const root = 0

// state is what a case did at a node.
type state uint8

const (
	// fresh is a node that no case has requested a choice at or ended at.
	fresh state = 0
	// request is a node where a case requested a choice.
	request state = 1
	// leaf is a node where a case ended.
	leaf state = 2
)

// node is one position of the tree.
type node struct {
	// bounds are the bounds of the request recorded at the node.
	bounds choice.Bounds
	// parent is the index of the node above. The root has none, and the
	// walk up from a leaf stops at the root before it reads the field.
	parent int32
	// state is what a case did at the node.
	state state
	// exhausted reports whether the node is a leaf, or whether every value
	// of its bounds leads to an exhausted node.
	exhausted bool
	// children is the number of values that cases took at the node.
	children uint64
	// exhaustedChildren is the number of those children that are
	// exhausted.
	exhaustedChildren uint64
}

// edge identifies a child: its parent and the value that a case took
// there. Floats compare by their bits, with one NaN, and a sequence by its
// elements.
type edge struct {
	// parent is the index of the node the value was taken at.
	parent int32
	// kind is the kind of the value.
	kind choice.Kind
	// integer is the value of an integer choice.
	integer choice.Int
	// bits are the bits of a float choice.
	bits uint64
	// sequence is the elements of a sequence choice, four little-endian
	// bytes each.
	sequence string
}

// Tree is the case tree of a run. Create one with [New].
type Tree struct {
	// nodes are every node, the root first.
	nodes []node
	// edges map each edge to the index of its child.
	edges map[edge]int32
	// limit is the most nodes the tree contains.
	limit int
	// full reports whether a walk found the tree at its limit.
	full bool
}

// New returns a tree of one root node that grows to limit nodes. A run's
// tree has a limit of [NodeLimit]. A limit of 1 or less adds no node to the
// root.
func New(limit int) *Tree {
	return &Tree{nodes: make([]node, 1), edges: make(map[edge]int32), limit: limit}
}

// Exhausted reports whether the run has tested every input of its domain:
// the root is exhausted and no walk found the tree at its limit.
func (t *Tree) Exhausted() bool {
	return !t.full && t.nodes[root].exhausted
}

// Full reports whether a walk found the tree at its limit.
func (t *Tree) Full() bool {
	return t.full
}

// Nodes returns the number of nodes in the tree, the root included.
func (t *Tree) Nodes() int {
	return len(t.nodes)
}

// Walk returns a walker for one new case, at the root.
func (t *Tree) Walk() *Walker {
	return &Walker{tree: t, path: []int32{root}}
}

// settle marks the node at index exhausted, and marks each node above it
// exhausted while all its values lead to exhausted nodes.
func (t *Tree) settle(index int32) {
	if t.nodes[index].exhausted {
		return
	}
	t.nodes[index].exhausted = true
	for index != root {
		index = t.nodes[index].parent
		n := &t.nodes[index]
		n.exhaustedChildren++
		count, finite := values(n.bounds)
		if !finite || n.children != count || n.exhaustedChildren != n.children {
			return
		}
		n.exhausted = true
	}
}

// values returns the number of values that b admits. It reports false for
// a float, for a sequence without a maximum length, and for a count of
// 2^64 or more, none of which a node exhausts.
func values(b choice.Bounds) (uint64, bool) {
	if b.Kind() == choice.Integer {
		count, carry := bits.Add64(b.Integer().Above(), b.Integer().Below(), 1)
		return count, carry == 0
	}
	if b.Kind() != choice.Sequence {
		return 0, false
	}
	sizes, k := b.Sequence().Sizes(), uint64(b.Sequence().K())
	maxSize, bounded := sizes.Max()
	if !bounded {
		return 0, false
	}
	lengths := maxSize - sizes.Min() + 1
	if k == 1 {
		return uint64(lengths), true
	}
	var total, carries uint64
	for i := range lengths {
		count, finite := power(k, sizes.Min()+i)
		if !finite {
			return 0, false
		}
		var carry uint64
		total, carry = bits.Add64(total, count, 0)
		carries |= carry
	}
	return total, carries == 0
}

// power returns k^n for a k of 2 or more. It reports false when the power
// is 2^64 or more, which it finds within 64 multiplications.
func power(k uint64, n int) (uint64, bool) {
	result := uint64(1)
	for range n {
		hi, lo := bits.Mul64(result, k)
		if hi != 0 {
			return 0, false
		}
		result = lo
	}
	return result, true
}
