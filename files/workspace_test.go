// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
)

// TestWorkspace checks the directory that a workspace writes.
func TestWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("Workspace", func(t *testing.T) {
		t.Parallel()

		t.Run("writes each entry, with 0644 and 0755 where the tree states no mode", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := files.Workspace(t, files.Tree{
				"go.mod":  files.Text("module example.com/a\n"),
				"bin/run": files.Executable("#!/bin/sh\n"),
				"keys/id": files.Text("secret\n").WithMode(privateMode),
				"current": files.Link("go.mod"),
			})
			content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			assert.NoError(t, err, "the file is there")
			assert.Equal(t, string(content), "module example.com/a\n", "the file's content")
			modes := map[string]os.FileMode{}
			for _, name := range []string{"go.mod", "bin", "bin/run", "keys/id"} {
				var info os.FileInfo
				info, err = os.Lstat(filepath.Join(dir, name))
				assert.NoError(t, err, name+" is there")
				modes[name] = info.Mode().Perm()
			}
			assert.Equal(t, modes, map[string]os.FileMode{
				"go.mod": fileMode, "bin": dirMode, "bin/run": dirMode, "keys/id": privateMode,
			}, "the modes are those stated, and the defaults where none is")
			target, err := os.Readlink(filepath.Join(dir, "current"))
			assert.NoError(t, err, "the link is there")
			assert.Equal(t, target, "go.mod", "the link's target as stated")
		})

		t.Run("writes a tree whose directory forbids its owner to write, which the test removes", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := files.Workspace(
				t,
				files.Tree{"locked": files.Dir().WithMode(0o500), "locked/a.txt": files.Text("a")},
			)
			info, err := os.Lstat(filepath.Join(dir, "locked"))
			assert.NoError(t, err, "the directory is there")
			assert.Equal(t, info.Mode().Perm(), os.FileMode(0o500), "the mode is set after the file below it")
		})

		t.Run("ends with a fault for a tree that breaks a rule, before it writes anything", func(t *testing.T) {
			t.Parallel()

			seat := &faultSeat{T: t}
			dir := files.Workspace(seat, files.Tree{"a": files.Text(""), "a/b": files.Text(""), "c": files.Text("")})
			f := seat.fault()
			assert.Equal(t, f.Op, "files.Workspace", "the operation")
			assert.Equal(t, f.Path, fault.Path{fault.Key("a/b")}, "the entry below a file")
			entries, err := os.ReadDir(dir)
			assert.NoError(t, err, "the directory is read")
			assert.Empty(t, entries, "the workspace writes nothing")
		})

		t.Run("ends with a fault for an entry that the file system refuses", func(t *testing.T) {
			t.Parallel()

			seat := &faultSeat{T: t}
			files.Workspace(seat, files.Tree{longName: files.Text("")})
			f := seat.fault()
			assert.Equal(t, f.Path, fault.Path{fault.Key(longName)}, "the entry that is refused")
			assert.Equal(t, f.Reason, "the entry cannot be written", "the reason")
		})
	})
}

// TestWorkspaceAllocs checks the allocation ceiling of Workspace. Each call
// makes a directory of the test's own.
func TestWorkspaceAllocs(t *testing.T) {
	alloctest.Check(t, workspaceCases(t))
}

// BenchmarkWorkspace measures Workspace.
func BenchmarkWorkspace(b *testing.B) {
	for _, c := range workspaceCases(b) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// workspaceCases returns a call of Workspace of a tree of one file on tb,
// with its allocation ceiling.
func workspaceCases(tb testing.TB) []alloctest.Case {
	tb.Helper()
	tree := files.Tree{"a.txt": files.Text("a\n")}
	return []alloctest.Case{
		{Name: "Workspace", Call: func(assert.TB) { kept = files.Workspace(tb, tree) }, Allocs: 48},
	}
}
