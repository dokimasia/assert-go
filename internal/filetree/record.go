// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package filetree

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/literal"
)

// MaxPaths is the most paths that a record lists.
const MaxPaths = 64

// The detail fields of the record of a comparison of trees.
const (
	// WantField is the tree of the wanted entries at the paths that differ.
	WantField = "want"
	// GotField is the tree of the entries read at those paths.
	GotField = "got"
	// DifferencesField is the number of paths that differ.
	DifferencesField = "differences"
)

// Record is the detail of a comparison of trees that fails: the wanted
// entries and the entries read at the first [MaxPaths] of the paths that
// differ, and the number of those paths. The record of a missing golden
// tree states no wanted tree: Want is nil.
type Record struct {
	// Want is the tree of the wanted entries at the paths that differ, and
	// nil for a missing golden tree.
	Want Tree
	// Got is the tree of the entries read at those paths.
	Got Tree
	// Differences is the number of paths that differ.
	Differences int
}

// NewRecord returns the record of a comparison of got with want, at paths,
// the paths that differ in path order. A missing entry is in Want alone, an
// extra one in Got alone, and a changed one in both. An entry of Got states
// its mode only where the entry of want at its path states one, and a file
// of Got that states none states whether its owner may execute it.
//
// # Allocation contract
//
// NewRecord allocates the wanted tree with its implied directories, and the
// two trees of the record.
func NewRecord(want, got Tree, paths []string) Record {
	wanted := want.Full()
	r := Record{Want: Tree{}, Got: Tree{}, Differences: len(paths)}
	for _, path := range paths[:min(len(paths), MaxPaths)] {
		if w, ok := wanted[path]; ok {
			r.Want[path] = w
		}
		if g, ok := got[path]; ok {
			r.Got[path] = asCompared(g, wanted[path])
		}
	}
	return r
}

// Missing returns the record of a golden tree that is missing: no wanted
// tree, the first [MaxPaths] entries of got in path order, each without its
// mode, and the number of the entries of got.
//
// # Allocation contract
//
// Missing allocates the paths of got and the tree of the record.
func Missing(got Tree) Record {
	paths := got.Paths()
	r := Record{Got: Tree{}, Differences: len(paths)}
	for _, path := range paths[:min(len(paths), MaxPaths)] {
		r.Got[path] = asCompared(got[path], Entry{})
	}
	return r
}

// asCompared returns read, an entry read where the wanted entry w is, as the
// comparison read it: with its mode where w states one, and with the
// execute bit of a file alone where w states none.
func asCompared(read, w Entry) Entry {
	read.Stated = read.Stated && w.Stated
	if !read.Stated {
		read.Mode &= executeBit(read.Kind)
	}
	return read
}

// Fields returns the detail of r as a failure states it: want, got and
// differences, with want nil for a missing golden tree.
//
// # Allocation contract
//
// Fields allocates the map and the interfaces of its values: four
// allocations.
func (r Record) Fields() map[string]any {
	var want any
	if r.Want != nil {
		want = r.Want
	}
	return map[string]any{WantField: want, GotField: r.Got, DifferencesField: r.Differences}
}

// recordJSON is the detail of a record as its call record states it.
type recordJSON struct {
	Want        json.RawMessage `json:"want"`
	Got         Tree            `json:"got"`
	Differences json.RawMessage `json:"differences"`
}

// MarshalJSON returns the detail of r as the call record of a comparison
// states it: the tree literal of want, or the literal of null for a missing
// golden tree, the tree literal of got, and the int literal of differences.
//
// # Allocation contract
//
// MarshalJSON allocates the literals of the two trees and the JSON that it
// returns.
func (r Record) MarshalJSON() ([]byte, error) {
	want := literal.Detail(nil)
	if r.Want != nil {
		// A tree of entries that a comparison read marshals.
		want, _ = r.Want.MarshalJSON()
	}
	return json.Marshal(recordJSON{Want: want, Got: r.Got, Differences: literal.Detail(r.Differences)})
}
