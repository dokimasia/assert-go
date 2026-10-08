// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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

	t.Run("Overwrite", func(t *testing.T) {
		t.Parallel()

		t.Run("creates each missing entry, replaces each file and link, and keeps the others", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := written(t, filetree.Tree{
				"a.txt":      fileOf("old\n"),
				"current":    linkTo("a.txt"),
				"docs":       dirWith(0o700),
				"docs/x.txt": fileWith("x\n", privateMode),
			})
			assert.NoError(t, filetree.Overwrite(dir, filetree.Tree{
				"a.txt":        fileOf("new\n"),
				"bin/run":      executableFile,
				"current":      linkTo("docs/x.txt"),
				"docs/new.txt": textFile,
			}), "the tree is written over the directory")
			assert.Equal(t, readBack(t, dir), filetree.Tree{
				"a.txt":        fileWith("new\n", fileMode),
				"bin":          dirWith(dirMode),
				"bin/run":      fileWith("#!/bin/sh\n", dirMode),
				"current":      linkTo("docs/x.txt"),
				"docs":         dirWith(0o700),
				"docs/new.txt": fileWith("a\n", fileMode),
				"docs/x.txt":   fileWith("x\n", privateMode),
			}, "a directory that the tree implies keeps its mode, and every entry that it does not state is kept")
		})

		t.Run("sets each stated mode, and the mode of a directory after every entry below it", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := written(t, filetree.Tree{"ro": directory, "ro/a.txt": textFile})
			assert.NoError(t, filetree.Overwrite(dir, filetree.Tree{
				"ro":           dirWith(0o500),
				"ro/a.txt":     fileWith("b\n", 0o400),
				"ro/new":       dirWith(0o500),
				"ro/new/b.txt": textFile,
			}), "the tree is written over the directory")
			assert.Equal(t, readBack(t, dir), filetree.Tree{
				"ro":           dirWith(0o500),
				"ro/a.txt":     fileWith("b\n", 0o400),
				"ro/new":       dirWith(0o500),
				"ro/new/b.txt": fileWith("a\n", fileMode),
			}, "each directory loses its write bit after the entries below it are written")
		})

		t.Run("sets the mode of each directory that is there after the directories below it", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)

			dir := written(t, filetree.Tree{"a/b": directory})
			err := filetree.Overwrite(dir, filetree.Tree{"a": dirWith(privateMode), "a/b": dirWith(0o700)})
			assert.NoError(t, err, "the directory below gets its mode while its parent may still be searched")
			info, err := os.Lstat(filepath.Join(dir, "a"))
			assert.NoError(t, err, "the parent is there")
			assert.Equal(t, info.Mode().Perm(), privateMode, "the parent's stated mode, without the search bit")
		})

		t.Run("replaces a file that its owner may not write", func(t *testing.T) {
			t.Parallel()

			dir := written(t, filetree.Tree{"ro.txt": fileWith("old\n", 0o444)})
			assert.NoError(t, filetree.Overwrite(dir, filetree.Tree{"ro.txt": fileOf("new\n")}), "the file is replaced")
			assert.Equal(t, readBack(t, dir)["ro.txt"].Content, "new\n", "the file's content")
			info, err := os.Stat(filepath.Join(dir, "ro.txt"))
			assert.NoError(t, err, "the file is there")
			assert.Equal(t, info.Mode().Perm()&ownerWrite, ownerWrite, "the file gets the owner's write bit of 0644")
		})

		t.Run("replaces a link, and keeps the entry that the link points to", func(t *testing.T) {
			t.Parallel()

			outside := written(t, filetree.Tree{"keep.txt": textFile})
			dir := written(t, filetree.Tree{"latest": linkTo(filepath.Join(outside, "keep.txt"))})
			assert.NoError(t, filetree.Overwrite(dir, filetree.Tree{"latest": linkTo("a.txt")}), "the link is replaced")
			assert.Equal(t, readBack(t, dir), filetree.Tree{"latest": linkTo("a.txt")}, "the link's new target")
			kept, err := filetree.Read(os.DirFS(outside), false)
			assert.NoError(t, err, "the directory outside is read")
			assert.Equal(t, kept, filetree.Tree{"keep.txt": textFile}, "the entry that the old target names is kept")
		})

		tests := []struct {
			name         string
			giveExisting filetree.Tree
			give         filetree.Tree
			wantPath     fault.Path
			wantReason   string
		}{
			{
				name:         "returns a fault at a file where the tree states a directory, and writes nothing",
				giveExisting: filetree.Tree{"a": textFile},
				give:         filetree.Tree{"0.txt": textFile, "a/b.txt": textFile},
				wantPath:     fault.Path{fault.Key("a")},
				wantReason:   "the entry is a file, and the tree states a directory",
			},
			{
				name:         "returns a fault at a directory where the tree states a file, and writes nothing",
				giveExisting: filetree.Tree{"a": directory},
				give:         filetree.Tree{"0.txt": textFile, "a": textFile},
				wantPath:     fault.Path{fault.Key("a")},
				wantReason:   "the entry is a directory, and the tree states a file",
			},
			{
				name:         "returns a fault at a link to a directory where the tree states a directory",
				giveExisting: filetree.Tree{"docs": directory, "a": linkTo("docs")},
				give:         filetree.Tree{"0.txt": textFile, "a/b.txt": textFile},
				wantPath:     fault.Path{fault.Key("a")},
				wantReason:   "the entry is a link, and the tree states a directory",
			},
			{
				name:         "returns a fault at a link where the tree states a file, and writes nothing through it",
				giveExisting: filetree.Tree{"a.txt": textFile, "b": linkTo("a.txt")},
				give:         filetree.Tree{"0.txt": textFile, "b": fileOf("new\n")},
				wantPath:     fault.Path{fault.Key("b")},
				wantReason:   "the entry is a link, and the tree states a file",
			},
			{
				name:         "returns a fault at an entry that cannot be read, and writes nothing",
				giveExisting: filetree.Tree{},
				give:         filetree.Tree{"0.txt": textFile, longName: textFile},
				wantPath:     fault.Path{fault.Key(longName)},
				wantReason:   "the entry cannot be read",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				dir := written(t, tt.giveExisting)
				before := readBack(t, dir)
				err := filetree.Overwrite(dir, tt.give)
				expectFault(t, err, tt.wantPath, tt.wantReason)
				assert.Equal(t, readBack(t, dir), before, "the directory is unchanged")
			})
		}

		t.Run("returns a fault at a second name of a stated entry, and writes nothing", func(t *testing.T) {
			t.Parallel()

			dir := written(t, filetree.Tree{"a.txt": textFile})
			assert.NoError(t, os.Link(filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")), "a second name")
			before := readBack(t, dir)
			err := filetree.Overwrite(dir, filetree.Tree{"a.txt": fileOf("x\n"), "b.txt": fileOf("y\n")})
			expectFault(t, err, fault.Path{fault.Key("b.txt")}, `the file system maps the path to the entry at "a.txt"`)
			assert.Equal(t, readBack(t, dir), before, "the directory is unchanged")
		})

		t.Run("returns a fault at an entry that is no file, directory or link", func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == "windows" {
				t.Skip("a root on Windows refuses the name of a device, so no read of one returns its kind")
			}

			device := filepath.Base(os.DevNull)
			err := filetree.Overwrite(filepath.Dir(os.DevNull), filetree.Tree{device + "/a.txt": textFile})
			expectFault(t, err, fault.Path{fault.Key(device)}, "the entry is no file, directory or link")
		})

		t.Run("returns the fault of a tree that breaks a rule", func(t *testing.T) {
			t.Parallel()

			err := filetree.Overwrite(t.TempDir(), filetree.Tree{"a": {}})
			expectFault(t, err, fault.Path{fault.Key("a")}, "the entry states no file, directory or link")
		})

		t.Run("returns a fault for a directory that cannot be opened", func(t *testing.T) {
			t.Parallel()

			err := filetree.Overwrite(filepath.Join(t.TempDir(), "missing"), filetree.Tree{"a.txt": textFile})
			expectFault(t, err, nil, "the directory cannot be opened")
		})

		writes := []struct {
			name string
			give filetree.Tree
			path string
		}{
			{
				name: "returns a fault at a file that a directory without the owner's write bit refuses",
				give: filetree.Tree{"ro/new.txt": textFile},
				path: "ro/new.txt",
			},
			{
				name: "returns a fault at a link that a directory without the owner's write bit refuses to replace",
				give: filetree.Tree{"ro/latest": linkTo("b.txt")},
				path: "ro/latest",
			},
		}
		for _, tt := range writes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				mustRecordModes(t)
				if os.Geteuid() == 0 {
					t.Skip("root writes into a read-only directory")
				}

				dir := written(t, filetree.Tree{"ro": dirWith(0o500), "ro/latest": linkTo("a.txt")})
				err := filetree.Overwrite(dir, tt.give)
				expectFault(t, err, fault.Path{fault.Key(tt.path)}, "the entry cannot be written")
				assert.ErrorIs(t, err, fs.ErrPermission, "the cause is the error of the file system")
			})
		}
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
		{Name: "Overwrite", Call: func(assert.TB) { errKept = filetree.Overwrite(dir, filetree.Tree{}) }, Allocs: 5},
		{Name: "Unlock", Call: func(assert.TB) { filetree.Unlock(dir) }, Allocs: 8},
	}
}
