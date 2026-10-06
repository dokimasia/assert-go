// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package files

import (
	"fmt"
	"io/fs"

	"go.dokimi.dev/assert/internal/filetree"
)

// Entry is a file, a directory or a symbolic link of a [Tree]. The zero
// Entry states nothing, and a tree that contains one is refused.
type Entry struct {
	e filetree.Entry
}

// Text returns a file whose content is text.
//
// # Allocation contract
//
// Text allocates nothing.
func Text(text string) Entry {
	return Entry{filetree.Entry{Kind: filetree.File, Content: text}}
}

// Bytes returns a file whose content is content. It copies content.
//
// # Allocation contract
//
// Bytes allocates the copy of content.
func Bytes(content []byte) Entry {
	return Text(string(content))
}

// Executable returns a file whose content is text, which its owner may
// execute.
//
// # Allocation contract
//
// Executable allocates nothing.
func Executable(text string) Entry {
	return Entry{filetree.Entry{Kind: filetree.File, Content: text, Mode: filetree.OwnerExecute}}
}

// Dir returns a directory. A tree needs to state a directory only when the
// directory is empty or when the tree states its mode, because every parent
// of an entry is a directory of the tree.
//
// # Allocation contract
//
// Dir allocates nothing.
func Dir() Entry {
	return Entry{filetree.Entry{Kind: filetree.Dir}}
}

// Link returns a symbolic link to target. The target is text that no rule of
// a path limits, inside or outside the tree, and nothing follows it. A tree
// reads a target with slashes on every platform, so a target separates its
// names with slashes.
//
// # Allocation contract
//
// Link allocates nothing.
func Link(target string) Entry {
	return Entry{filetree.Entry{Kind: filetree.Link, Target: target}}
}

// WithMode returns e with the nine permission bits of mode, which a workspace
// sets and a comparison compares. A file's mode states whether its owner may
// execute it.
//
//	files.Text("secret\n").WithMode(0o600)
//
// # Panics
//
// It panics for a mode with a bit beyond the nine permission bits, for a
// link, which has no mode, and for the zero Entry, which states no entry.
//
// # Allocation contract
//
// WithMode allocates nothing.
func (e Entry) WithMode(mode fs.FileMode) Entry {
	if mode&^fs.ModePerm != 0 {
		panic(fmt.Sprintf("files: WithMode(%O) states a bit beyond the nine permission bits", uint32(mode)))
	}
	if e.e.Kind != filetree.File && e.e.Kind != filetree.Dir {
		panic(fmt.Sprintf("files: WithMode(%O) states the mode of no file and no directory", uint32(mode)))
	}
	e.e.Mode, e.e.Stated = mode, true
	return e
}
