// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package tree

import (
	"encoding/binary"
	"math"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// Walker follows one case at a time down a [Tree]. Create one with
// [Tree.Walk], call Step for each choice of the case and End when the case
// ends, and Restart it for the next case.
type Walker struct {
	// tree is the tree the walker follows.
	tree *Tree
	// path are the indices of the nodes from the root to the case's next
	// position.
	path []int32
	// off reports whether the walk found the tree at its limit, after
	// which it checks nothing for the rest of its case.
	off bool
	// key is the buffer a sequence's edge key is written in.
	key []byte
}

// Restart returns w to the root of its tree for one new case, as a walker
// that [Tree.Walk] returns, and keeps the storage of its path. It allocates
// nothing.
func (w *Walker) Restart() {
	w.path, w.off = append(w.path[:0], root), false
}

// Step follows the choice that the case took with bounds.
//
// It returns [ErrRepeated] when the choice arrives at a leaf, and a
// [*DivergenceError] when an earlier case ended at the position or
// requested other bounds there. At the limit of the tree it records
// nothing, and it returns nil for the rest of the case.
func (w *Walker) Step(bounds choice.Bounds, value choice.Choice) error {
	if w.off {
		return nil
	}
	t, at := w.tree, w.path[len(w.path)-1]
	index := len(w.path) - 1
	if t.nodes[at].state == leaf {
		return &DivergenceError{Index: index, Requested: new(bounds)}
	}
	if t.nodes[at].state == fresh {
		t.nodes[at].bounds, t.nodes[at].state = bounds, request
	}
	if t.nodes[at].bounds != bounds {
		return &DivergenceError{Index: index, Recorded: new(t.nodes[at].bounds), Requested: new(bounds)}
	}
	key := w.edge(at, value)
	child, found := t.edges[key]
	if !found {
		if len(t.nodes) >= t.limit {
			t.full, w.off = true, true
			return nil
		}
		child = int32(len(t.nodes))
		t.nodes = append(t.nodes, node{parent: at})
		t.nodes[at].children++
		t.edges[key] = child
	}
	if t.nodes[child].state == leaf {
		return ErrRepeated
	}
	w.path = append(w.path, child)
	return nil
}

// End marks the case's last position as a leaf, and marks each node above
// it exhausted while all its values lead to exhausted nodes.
//
// It returns a [*DivergenceError] when an earlier case requested a choice
// at that position. After the walk found the tree at its limit, it records
// nothing and returns nil.
func (w *Walker) End() error {
	if w.off {
		return nil
	}
	t, at := w.tree, w.path[len(w.path)-1]
	if t.nodes[at].state == request {
		return &DivergenceError{Index: len(w.path) - 1, Recorded: new(t.nodes[at].bounds)}
	}
	t.nodes[at].state = leaf
	t.settle(at)
	return nil
}

// edge returns the key of the value taken at the node at index.
func (w *Walker) edge(at int32, value choice.Choice) edge {
	key := edge{parent: at, kind: value.Kind}
	if value.Kind == choice.Integer {
		key.integer = value.Integer
		return key
	}
	if value.Kind == choice.Float {
		key.bits = math.Float64bits(value.Float)
		if math.IsNaN(value.Float) {
			key.bits = choice.NaNBits
		}
		return key
	}
	w.key = w.key[:0]
	for _, element := range value.Sequence {
		w.key = binary.LittleEndian.AppendUint32(w.key, element)
	}
	key.sequence = string(w.key)
	return key
}
