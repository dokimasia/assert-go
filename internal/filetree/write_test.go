// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// TestWrite checks the writes of a workspace and of an update of a golden
// tree, and the unlocking of a directory for its removal.
func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		t.Run("writes every entry with its stated mode, and 0644 and 0755 where it states none", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := written(t, filetree.Tree{
				"a.txt":   textFile,
				"bin/run": executableFile,
				"cache":   directory,
				"current": linkTo("a.txt"),
				"keys":    dirWith(0o700),
				"keys/id": fileWith("secret\n", privateMode),
				"up":      linkTo("../outside/missing.txt"),
			})
			assert.Equal(t, readBack(t, dir), filetree.Tree{
				"a.txt":   fileWith("a\n", fileMode),
				"bin":     dirWith(dirMode),
				"bin/run": fileWith("#!/bin/sh\n", dirMode),
				"cache":   dirWith(dirMode),
				"current": linkTo("a.txt"),
				"keys":    dirWith(0o700),
				"keys/id": fileWith("secret\n", privateMode),
				"up":      linkTo("../outside/missing.txt"),
			}, "a link's target is written as stated, outside the directory included")
		})

		t.Run("sets the mode of a directory after every entry below it", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := written(t, filetree.Tree{"ro": dirWith(0o500), "ro/a.txt": textFile})
			assert.Equal(
				t,
				readBack(t, dir),
				filetree.Tree{"ro": dirWith(0o500), "ro/a.txt": fileWith("a\n", fileMode)},
				"the file below the directory is written before the directory loses its write bit",
			)
		})

		t.Run("sets the owner's write bit of each file, the one bit that Windows records", func(t *testing.T) {
			t.Parallel()

			dir := written(t, filetree.Tree{"ro.txt": fileWith("a\n", 0o444), "rw.txt": fileWith("a\n", fileMode)})
			writable := map[string]bool{}
			for _, name := range []string{"ro.txt", "rw.txt"} {
				info, err := os.Stat(filepath.Join(dir, name))
				assert.NoError(t, err, name+" is there")
				writable[name] = info.Mode().Perm()&ownerWrite != 0
			}
			assert.Equal(t, writable, map[string]bool{"ro.txt": false, "rw.txt": true},
				"a file whose mode lacks the owner's write bit is read-only")
		})

		t.Run("returns the fault of a tree that breaks a rule, and writes nothing", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			err := filetree.Write(dir, filetree.Tree{"a": textFile, "a/b": textFile, "c": textFile})
			expectFault(t, err, fault.Path{fault.Key("a/b")}, `the entry is below "a", which is a file`)
			assert.Empty(t, readBack(t, dir), "the directory is empty")
		})

		t.Run("returns a fault at an entry that is there, and writes over none", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			assert.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("kept"), privateMode), "a file is there")
			err := filetree.Write(dir, filetree.Tree{"a.txt": fileOf("new")})
			expectFault(t, err, fault.Path{fault.Key("a.txt")}, "the entry cannot be written")
			assert.ErrorIs(t, err, fs.ErrExist, "the cause states that the entry is there")
			assert.Equal(t, readBack(t, dir)["a.txt"].Content, "kept", "the file is unchanged")
		})

		t.Run("returns a fault for a directory that cannot be opened", func(t *testing.T) {
			t.Parallel()

			err := filetree.Write(filepath.Join(t.TempDir(), "missing"), filetree.Tree{"a.txt": textFile})
			expectFault(t, err, nil, "the directory cannot be opened")
		})

		tests := []struct {
			name string
			give filetree.Entry
		}{
			{name: "returns a fault at a file that the file system refuses", give: textFile},
			{name: "returns a fault at a directory that the file system refuses", give: directory},
			{name: "returns a fault at a link that the file system refuses", give: linkTo("a.txt")},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := filetree.Write(t.TempDir(), filetree.Tree{longName: tt.give})
				expectFault(t, err, fault.Path{fault.Key(longName)}, "the entry cannot be written")
			})
		}
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()

		t.Run(
			"creates a missing directory and writes the tree with the modes that a workspace sets",
			func(t *testing.T) {
				t.Parallel()
				mustRecordModes(t)

				dir := filepath.Join(t.TempDir(), "testdata", "golden", "api")
				tree := filetree.Tree{"a.txt": fileWith("a\n", privateMode), "bin/run": executableFile}
				assert.NoError(t, filetree.Update(dir, tree), "the update writes the tree")
				assert.Equal(t, readBack(t, dir), filetree.Tree{
					"a.txt": fileWith(
						"a\n",
						fileMode,
					),
					"bin":     dirWith(dirMode),
					"bin/run": fileWith("#!/bin/sh\n", dirMode),
				}, "the update writes no stated mode")
			},
		)

		t.Run("removes each extra entry, rewrites a changed one, and follows no link", func(t *testing.T) {
			t.Parallel()

			outside := written(t, filetree.Tree{"keep.txt": textFile})
			dir := written(t, filetree.Tree{
				"api.go":       fileOf("package api\n"),
				"latest":       linkTo(filepath.Join(outside, "keep.txt")),
				"old/stale.go": fileOf("package old\n"),
				"stale.go":     fileOf("package api\n"),
			})
			tree := filetree.Tree{"api.go": fileOf("package api\n\nfunc New() {}\n"), "new.go": fileOf("package api\n")}
			assert.NoError(t, filetree.Update(dir, tree), "the update rewrites the directory")
			got, err := filetree.Read(os.DirFS(dir), false)
			assert.NoError(t, err, "the directory is read")
			assert.Equal(t, got, tree, "the directory equals the tree")
			kept, err := filetree.Read(os.DirFS(outside), false)
			assert.NoError(t, err, "the directory outside is read")
			assert.Equal(
				t,
				kept,
				filetree.Tree{"keep.txt": textFile},
				"the entry that the removed link points to stays",
			)
		})

		t.Run("leaves an entry that equals the tree's as it is, its mode included", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := written(t, filetree.Tree{"a.txt": fileWith("a\n", privateMode)})
			assert.NoError(t, filetree.Update(dir, filetree.Tree{"a.txt": textFile}), "the update passes")
			assert.Equal(t, readBack(t, dir), filetree.Tree{"a.txt": fileWith("a\n", privateMode)},
				"the file keeps its mode, which the update does not compare")
		})

		t.Run("returns the fault of a tree that breaks a rule", func(t *testing.T) {
			t.Parallel()

			err := filetree.Update(t.TempDir(), filetree.Tree{"a": {}})
			expectFault(t, err, fault.Path{fault.Key("a")}, "the entry states no file, directory or link")
		})

		t.Run("returns a fault for a directory that cannot be created", func(t *testing.T) {
			t.Parallel()

			dir := written(t, filetree.Tree{"a.txt": textFile})
			err := filetree.Update(filepath.Join(dir, "a.txt", "golden"), filetree.Tree{"b": textFile})
			expectFault(t, err, nil, "the directory cannot be created")
		})

		t.Run("returns the fault of a directory that cannot be read", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := written(t, filetree.Tree{"locked": dirWith(0o300)})
			err := filetree.Update(dir, filetree.Tree{"a.txt": textFile})
			expectFault(t, err, nil, "the tree cannot be read")
		})

		t.Run("returns a fault at an entry that cannot be written", func(t *testing.T) {
			t.Parallel()

			err := filetree.Update(t.TempDir(), filetree.Tree{longName: textFile})
			expectFault(t, err, fault.Path{fault.Key(longName)}, "the entry cannot be written")
		})
	})

	t.Run("Unlock", func(t *testing.T) {
		t.Parallel()

		t.Run("gives the owner every permission of each directory, so the tree can be removed", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := t.TempDir()
			assert.NoError(t, filetree.Write(dir, filetree.Tree{"locked": dirWith(0), "locked/a.txt": textFile}),
				"the tree is written")
			filetree.Unlock(dir)
			assert.Equal(t, readBack(t, dir)["locked"], dirWith(0o700), "the owner may read, write and search it")
			assert.NoError(t, os.RemoveAll(dir), "the tree is removed")
		})
	})
}

// TestWriteAllocs checks the allocation ceilings of the writers of a tree.
// It writes into a directory, which a test that runs alone measures.
func TestWriteAllocs(t *testing.T) {
	alloctest.Check(t, writeCases(t))
}

// BenchmarkWrite measures the writers of a tree.
func BenchmarkWrite(b *testing.B) {
	for _, c := range writeCases(b) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// writeCases returns a call of each writer of an empty tree into an empty
// directory, which writes nothing, with its allocation ceiling, measured.
func writeCases(tb testing.TB) []alloctest.Case {
	tb.Helper()
	dir := tb.TempDir()
	return []alloctest.Case{
		{Name: "Write", Call: func(assert.TB) { errKept = filetree.Write(dir, filetree.Tree{}) }, Allocs: 5},
		{Name: "Update", Call: func(assert.TB) { errKept = filetree.Update(dir, filetree.Tree{}) }, Allocs: 18},
		{Name: "Unlock", Call: func(assert.TB) { filetree.Unlock(dir) }, Allocs: 8},
	}
}
