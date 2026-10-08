// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
)

// readOp is the operation of Read, which names its faults.
const readOp = "files.Read"

// Read returns the content of the file at path, a path of the operating
// system, so that a test composes it with the text assertions:
//
//	assert.Contains(t, files.Read(t, path), "version 2", "the file states the new version")
//
// # Errors
//
// It ends the call with a fault, which stops the test, when no file is at
// path, a link to a file included, and when the file cannot be read. It
// returns the empty string then, on a seat whose Fatalf returns.
//
// # Allocation contract
//
// Read allocates what os.Lstat and os.ReadFile allocate, and the content as
// the text that it returns: 8 allocations for a short file.
func Read(tb assert.TB, path string) string {
	tb.Helper()
	e, err := filetree.ReadPath(path, true)
	if err == nil && e.Kind != filetree.File {
		err = fault.New("no file is at the path %q", path)
	}
	if err != nil {
		matcher.End(tb, fault.In(readOp, err))
		return ""
	}
	return e.Content
}
