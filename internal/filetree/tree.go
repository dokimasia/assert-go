// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree

import (
	"io/fs"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/fault"
)

// Tree is a tree of files: each entry by its path, relative to the tree's
// root and separated by slashes. The directories of a tree are the ones it
// states and every parent of an entry.
type Tree map[string]Entry

// CheckPath returns a fault for a path that breaks a rule of a path, and nil
// for one that keeps them. A path is UTF-8 text of one or more names joined
// by a slash. A name is not empty, . or .., and contains no backslash and no
// NUL.
//
// # Allocation contract
//
// CheckPath allocates nothing for a path that keeps the rules.
func CheckPath(path string) error {
	if !utf8.ValidString(path) {
		return fault.New("the path %q is no UTF-8 text", path)
	}
	for name := range strings.SplitSeq(path, "/") {
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "\\\x00") {
			return fault.New("the path %q has the name %q", path, name)
		}
	}
	return nil
}

// Check returns a fault at the first entry of t, in path order, that breaks
// a rule of a tree, and nil for a tree that keeps them. A path keeps the
// rules of [CheckPath]. An entry is no zero Entry, a mode has no bit beyond
// the nine permission bits, and a link states no mode and a target of text
// without NUL that is not empty. No entry is below a file or a link.
//
// # Allocation contract
//
// Check allocates the paths in order, once, for a tree that keeps the
// rules.
func (t Tree) Check() error {
	for _, path := range t.Paths() {
		if err := t.checkEntry(path); err != nil {
			return fault.At(err, fault.Key(path))
		}
	}
	return nil
}

// checkEntry returns a fault for the entry at path when it breaks a rule of
// a tree.
func (t Tree) checkEntry(path string) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	e := t[path]
	if e.Kind < File || e.Kind > Link {
		return fault.New("the entry states no file, directory or link")
	}
	if e.Mode&^fs.ModePerm != 0 {
		return fault.New("the mode %O has a bit beyond the nine permission bits", uint32(e.Mode))
	}
	target := e.Target != "" && !strings.ContainsRune(e.Target, 0) && utf8.ValidString(e.Target)
	if e.Kind == Link && (e.Stated || !target) {
		return fault.New("the link to %q states a mode, or no target of UTF-8 text without NUL", e.Target)
	}
	for parent := range parents(path) {
		if kind := t[parent].Kind; kind == File || kind == Link {
			return fault.New("the entry is below %q, which is a %s", parent, kind.Name())
		}
	}
	return nil
}

// Paths returns the paths of t in path order, the order of their bytes.
//
// # Allocation contract
//
// Paths allocates the list once.
func (t Tree) Paths() []string {
	out := make([]string, 0, len(t))
	for path := range t {
		out = append(out, path)
	}
	slices.Sort(out)
	return out
}

// Full returns a copy of t that states every directory that t implies, a
// parent of an entry that t does not state, without a mode.
//
// # Allocation contract
//
// Full allocates the copy, and its growth by the directories it adds.
func (t Tree) Full() Tree {
	out := maps.Clone(t)
	for path := range t {
		for parent := range parents(path) {
			if _, stated := out[parent]; !stated {
				out[parent] = Entry{Kind: Dir}
			}
		}
	}
	return out
}

// parents yields each directory above path, outermost first: a, then a/b,
// for a/b/c.
func parents(path string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for i := range len(path) {
			if path[i] == '/' && !yield(path[:i]) {
				return
			}
		}
	}
}
