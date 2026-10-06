// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package files

import (
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// The operations of the methods of a tree, which name their faults.
const (
	marshalOp   = "files.Tree.MarshalJSON"
	unmarshalOp = "files.Tree.UnmarshalJSON"
)

// Tree is a tree of files: each entry at its path, relative to the tree's
// root and separated by slashes. A path is one or more names, none of them
// empty, . or .., and none containing a backslash or NUL. No entry is below
// a file or a link.
//
// Tree implements json.Marshaler and json.Unmarshaler with the tree literal
// of the definition: {"type": "tree", "entries": [...]}, its entries in the
// order of their paths' bytes.
type Tree map[string]Entry

// internal returns t as the package filetree states a tree.
func (t Tree) internal() filetree.Tree {
	out := make(filetree.Tree, len(t))
	for path, entry := range t {
		out[path] = entry.e
	}
	return out
}

// MarshalJSON returns the tree literal of t, which [Tree.UnmarshalJSON]
// reads back. A file states its whole content, however long, as text when
// the content is valid UTF-8, and as lowercase hexadecimal bytes otherwise.
// A file that states no mode states executable when its owner may execute
// it.
//
// # Errors
//
// It returns a fault at the first entry, in path order, that breaks a rule of
// a tree.
//
// # Allocation contract
//
// MarshalJSON allocates the tree that it converts t to, the literal of each
// entry, and the JSON that it returns: 14 allocations for a tree of six
// entries of every kind.
func (t Tree) MarshalJSON() ([]byte, error) {
	internal := t.internal()
	if err := internal.Check(); err != nil {
		return nil, fault.In(marshalOp, err)
	}
	return internal.Encode()
}

// UnmarshalJSON sets t to the tree that the tree literal data states.
//
// # Errors
//
// It returns a fault for a text that is no tree literal, whose path leads
// through the literal's JSON to what it misstates, such as entries[2].mode,
// and for a tree that breaks a rule of a tree. A file stated by its digest,
// which only a record states, is no entry of an input. It leaves t unchanged
// then.
//
// # Allocation contract
//
// UnmarshalJSON allocates what the JSON decoder allocates for data, and the
// tree: 122 allocations for the literal of six entries of every kind.
func (t *Tree) UnmarshalJSON(data []byte) error {
	decoded, err := filetree.Decode(data)
	if err != nil {
		return fault.In(unmarshalOp, err)
	}
	out := make(Tree, len(decoded))
	for path, e := range decoded {
		out[path] = Entry{e}
	}
	*t = out
	return nil
}
