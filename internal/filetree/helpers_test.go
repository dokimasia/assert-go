// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// The values that the measured calls return. The compiler keeps a call
// whose result a test stores here.
var (
	kept       string
	keptBool   bool
	errKept    error
	keptTree   filetree.Tree
	keptPaths  []string
	keptBytes  []byte
	keptRecord filetree.Record
	keptFields map[string]any
)

// The entries that more than one test states.
var (
	// textFile is a file whose content is "a\n". Its mode is not stated.
	textFile = filetree.Entry{Kind: filetree.File, Content: "a\n"}
	// executableFile is a script that its owner may execute.
	executableFile = filetree.Entry{Kind: filetree.File, Content: "#!/bin/sh\n", Mode: filetree.OwnerExecute}
	// directory is a directory whose mode is not stated.
	directory = filetree.Entry{Kind: filetree.Dir}
)

// The modes of the files and directories that the tests read and write.
const (
	// fileMode is the default mode of a file that a workspace writes.
	fileMode fs.FileMode = 0o644
	// dirMode is the default mode of a directory that a workspace writes.
	dirMode fs.FileMode = 0o755
	// privateMode is the mode of a file that only its owner may read and
	// write.
	privateMode fs.FileMode = 0o600
	// ownerWrite is the permission bit that lets the owner write a file.
	// Windows records this bit in the read-only attribute of the file.
	ownerWrite fs.FileMode = 0o200
)

// fileOf returns a file whose content is content. Its mode is not stated.
func fileOf(content string) filetree.Entry {
	return filetree.Entry{Kind: filetree.File, Content: content}
}

// fileWith returns a file whose content is content. Its stated mode is mode.
func fileWith(content string, mode fs.FileMode) filetree.Entry {
	return filetree.Entry{Kind: filetree.File, Content: content, Mode: mode, Stated: true}
}

// dirWith returns a directory whose stated mode is mode.
func dirWith(mode fs.FileMode) filetree.Entry {
	return filetree.Entry{Kind: filetree.Dir, Mode: mode, Stated: true}
}

// linkTo returns a link to target.
func linkTo(target string) filetree.Entry {
	return filetree.Entry{Kind: filetree.Link, Target: target}
}

// mustRecordModes skips tb on a platform whose file systems do not record
// permission bits. The entries that Read returns there do not have a mode
// or an execute bit.
func mustRecordModes(tb testing.TB) {
	tb.Helper()
	if err := filetree.ModesUnrecorded(); err != nil {
		tb.Skip(err)
	}
}

// expectFault checks that err is a fault at path whose reason is reason.
func expectFault(t *testing.T, err error, path fault.Path, reason string) {
	t.Helper()

	f := assert.ErrorAs[*fault.Error](t, err, "a fault")
	assert.Equal(t, f.Path, path, "the path of the fault")
	assert.Equal(t, f.Reason, reason, "the reason of the fault")
}

// written writes tree into a fresh directory, and returns the directory.
func written(t *testing.T, tree filetree.Tree) string {
	t.Helper()

	dir := t.TempDir()
	t.Cleanup(func() { filetree.Unlock(dir) })
	assert.NoError(t, filetree.Write(dir, tree), "the tree is written")
	return dir
}

// readBack returns the tree in dir. The tree includes the mode of each entry.
func readBack(t *testing.T, dir string) filetree.Tree {
	t.Helper()

	tree, err := filetree.Read(os.DirFS(dir), true)
	assert.NoError(t, err, "the tree is read back")
	return tree
}

// longName is a name longer than any file system accepts in one entry.
var longName = strings.Repeat("n", 300)

// failingFS is a file system with two entries that fail. Info returns
// errFailing for the entry named info. ReadFile returns an error that wraps
// errFailing for the file named read. Every other entry is the entry of the
// map that failingFS embeds.
type failingFS struct {
	fstest.MapFS

	info string
	read string
}

// errFailing is the error that the failing entries of a failingFS return.
var errFailing = errors.New("filetree: the entry fails")

// ReadFile reads the file name, and fails for the file named read.
func (f failingFS) ReadFile(name string) ([]byte, error) {
	if name == f.read {
		return nil, &fs.PathError{Op: "read", Path: name, Err: errFailing}
	}
	return f.MapFS.ReadFile(name)
}

// ReadDir reads the directory name. It replaces the entry named info with a
// failingEntry.
func (f failingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := f.MapFS.ReadDir(name)
	for i, e := range entries {
		if e.Name() == f.info {
			entries[i] = failingEntry{e}
		}
	}
	return entries, err
}

// failingEntry is a directory entry whose Info method returns errFailing.
type failingEntry struct {
	fs.DirEntry
}

// Info returns errFailing.
func (failingEntry) Info() (fs.FileInfo, error) {
	return nil, errFailing
}

// linklessFS is a file system that implements fs.FS alone. A caller cannot
// read a link through it.
type linklessFS struct {
	fsys fs.FS
}

// Open opens name in the file system that it wraps.
func (f linklessFS) Open(name string) (fs.File, error) {
	return f.fsys.Open(name)
}
