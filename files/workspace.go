// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package files

import (
	"testing"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
)

// workspaceOp is the operation of Workspace, which names its faults.
const workspaceOp = "files.Workspace"

// Workspace writes tree into a directory of the test's own, which the test
// removes when it ends, and returns the directory's path.
//
// It creates every entry and writes over none. It writes through os.Root, so
// no entry is written outside the directory, and it never follows a link: it
// writes a link's target as the tree states it, inside or outside the
// directory. Each file and each directory gets its stated mode. Where the
// tree states none, a file gets 0644, and a file that its owner may execute
// and a directory get 0755. Workspace sets each mode after it writes the
// entry, and the mode of a directory after every entry below it, so the
// umask cannot change a mode, and the tree that it writes is the same on
// every run. On Windows only the owner's write bit of a file's mode takes
// effect, as the file's read-only attribute. The test removes the directory
// also when the mode of a directory in it forbids the owner to write it.
//
// It takes a testing.TB, whose TempDir it writes the tree into.
//
// # Errors
//
// A tree that breaks a rule of a tree ends the call with a fault before
// Workspace writes anything, and so does an entry that the file system
// cannot store as stated, such as a link where the platform refuses to
// create one, a name that the platform reserves, or a second path that the
// file system maps to an entry already written. The fault stops the test.
//
// # Allocation contract
//
// Workspace allocates the test's directory, the tree that it writes with its
// implied directories, an operation for each entry, and what the file
// system's calls allocate. A workspace of one short file allocates 28 times.
func Workspace(tb testing.TB, tree Tree) string {
	tb.Helper()
	dir := tb.TempDir()
	tb.Cleanup(func() { filetree.Unlock(dir) })
	if err := filetree.Write(dir, tree.internal()); err != nil {
		matcher.End(tb, fault.In(workspaceOp, err))
	}
	return dir
}
