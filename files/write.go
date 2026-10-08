// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
)

// writeOp is the operation of Write, which names its faults.
const writeOp = "files.Write"

// Write writes tree into dir, a directory that exists. Where nothing is at
// a path of the tree, it creates the entry as [Workspace] does. A file
// replaces the content of the file at its path, a link replaces the target
// of the link at its path, and a directory keeps its entries. Each entry
// that the tree states gets its stated mode, and where it states none 0644,
// or 0755 for an executable file and a directory. A parent that the tree
// implies keeps its mode when it exists, and gets 0755 when Write creates
// it. Every other entry of dir keeps its content and its mode.
//
// It writes through os.Root and never follows a link: it replaces a link by
// removing the link itself. A test that edits a file between two calls of
// the code under test writes the edit with it:
//
//	dir := files.Workspace(t, files.Tree{"api/store.gen.go": files.Text(generated)})
//	assert.NoError(t, gen.Run(dir), "the first run writes the file")
//	files.Write(t, dir, files.Tree{"api/store.gen.go": files.Text(edited)})
//	assert.ErrorIs(t, gen.Run(dir), gen.ErrDrift, "the second run reports the edit")
//
// It takes an assert.TB, because it creates no directory of the test's own.
//
// # Errors
//
// A tree that breaks a rule of a tree, a dir that is no directory, an entry
// of another kind than the tree states, a parent of an entry that is no
// directory, and a path that the file system maps to an entry at another
// path of the tree each end the call with a fault before Write writes
// anything. A link is another kind than a directory, a link to a directory
// included. An error of the file system ends the call with a fault at the
// path of its entry, such as a file that a directory without the owner's
// write bit refuses. The fault stops the test.
//
// # Allocation contract
//
// Write allocates the tree that it writes with its implied directories, the
// information of each entry that it reads, an operation for each entry, and
// what the file system's calls allocate. A write of one short file over a
// file allocates 28 times.
func Write(tb assert.TB, dir string, tree Tree) {
	tb.Helper()
	if err := filetree.Overwrite(dir, tree.internal()); err != nil {
		matcher.End(tb, fault.In(writeOp, err))
	}
}
