// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// mapped is a file system in memory of an entry of every kind, with modes.
var mapped = fstest.MapFS{
	"docs/a.md": {Data: []byte("# a\n"), Mode: 0o640},
	"docs":      {Mode: fs.ModeDir | 0o750},
	"run.sh":    {Data: []byte("#!/bin/sh\n"), Mode: 0o755},
	"current":   {Data: []byte("docs/a.md"), Mode: fs.ModeSymlink | 0o777},
}

// TestRead checks the trees that a file system reads as, and the entry at
// one path of the operating system.
func TestRead(t *testing.T) {
	t.Parallel()

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		t.Run("returns every entry with its permission bits, and a link as a link", func(t *testing.T) {
			t.Parallel()

			tree, err := filetree.Read(mapped, true)
			assert.NoError(t, err, "the tree is read")
			assert.Equal(t, tree, filetree.Tree{
				"docs/a.md": fileWith("# a\n", 0o640),
				"docs":      dirWith(0o750),
				"run.sh":    fileWith("#!/bin/sh\n", 0o755),
				"current":   linkTo("docs/a.md"),
			}, "a link states its target and no mode")
		})

		t.Run("returns the execute bit of each file alone without modes", func(t *testing.T) {
			t.Parallel()

			tree, err := filetree.Read(mapped, false)
			assert.NoError(t, err, "the tree is read")
			assert.Equal(t, tree, filetree.Tree{
				"docs/a.md": fileOf("# a\n"),
				"docs":      directory,
				"run.sh":    executableFile,
				"current":   linkTo("docs/a.md"),
			}, "a directory states no bit")
		})

		t.Run("returns the entries of a directory, and follows no link", func(t *testing.T) {
			t.Parallel()

			dir := written(t, filetree.Tree{"a.txt": textFile, "up": linkTo("..")})
			assert.Equal(t, readBack(t, dir), filetree.Tree{"a.txt": fileWith("a\n", fileMode), "up": linkTo("..")},
				"the link to the parent is read as a link")
		})

		t.Run("returns a fault for a root that is no directory", func(t *testing.T) {
			t.Parallel()

			_, err := filetree.Read(fstest.MapFS{".": {Data: []byte("a\n")}}, true)
			expectFault(t, err, nil, "the root of the tree is no directory")
		})

		t.Run("returns a fault whose cause is the error of the file system for a missing root", func(t *testing.T) {
			t.Parallel()

			_, err := filetree.Read(os.DirFS(filepath.Join(t.TempDir(), "missing")), true)
			expectFault(t, err, nil, "the tree cannot be read")
			assert.ErrorIs(t, err, fs.ErrNotExist, "the cause states that the root is missing")
		})

		tests := []struct {
			name       string
			give       fs.FS
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault at an entry that is no file, directory or link",
				give:       fstest.MapFS{"pipe": {Mode: fs.ModeNamedPipe}},
				wantPath:   fault.Path{fault.Key("pipe")},
				wantReason: "the entry is no file, directory or link",
			},
			{
				name:       "returns a fault at a link of a file system that reads no link",
				give:       linklessFS{fstest.MapFS{"l": {Data: []byte("a"), Mode: fs.ModeSymlink}}},
				wantPath:   fault.Path{fault.Key("l")},
				wantReason: "the entry cannot be read",
			},
			{
				name:       "returns a fault at an entry whose information cannot be read",
				give:       failingFS{MapFS: fstest.MapFS{"a": {Data: []byte("a")}}, info: "a"},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: "the entry cannot be read",
			},
			{
				name:       "returns a fault at a file that cannot be read",
				give:       failingFS{MapFS: fstest.MapFS{"a": {Data: []byte("a")}}, read: "a"},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: "the entry cannot be read",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := filetree.Read(tt.give, true)
				expectFault(t, err, tt.wantPath, tt.wantReason)
			})
		}
	})

	t.Run("ReadPath", func(t *testing.T) {
		t.Parallel()

		dir := written(t, filetree.Tree{
			"a.txt":   fileWith("a\n", privateMode),
			"keys":    dirWith(0o700),
			"current": linkTo("a.txt"),
		})

		tests := []struct {
			name    string
			path    string
			content bool
			want    filetree.Entry
		}{
			{
				name: "returns a file with its mode and without its content", path: "a.txt",
				want: filetree.Entry{Kind: filetree.File, Mode: privateMode, Stated: true},
			},
			{
				name: "returns a file with its content when asked", path: "a.txt", content: true,
				want: fileWith("a\n", privateMode),
			},
			{name: "returns a directory with its mode", path: "keys", want: dirWith(0o700)},
			{
				name: "returns a link with its target, without following it", path: "current", content: true,
				want: linkTo("a.txt"),
			},
			{name: "returns no entry where nothing is", path: "b.txt", want: filetree.Entry{}},
			{name: "returns no entry below a file", path: "a.txt/b", want: filetree.Entry{}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := filetree.ReadPath(filepath.Join(dir, tt.path), tt.content)
				assert.NoError(t, err, "the entry is read")
				assert.Equal(t, got, tt.want, "the entry at the path")
			})
		}

		t.Run("returns a fault for an entry that is no file, directory or link", func(t *testing.T) {
			t.Parallel()

			_, err := filetree.ReadPath(os.DevNull, false)
			expectFault(t, err, nil, "the entry is no file, directory or link")
		})

		t.Run("returns a fault whose cause is the error of the file system for a name too long", func(t *testing.T) {
			t.Parallel()

			_, err := filetree.ReadPath(filepath.Join(t.TempDir(), longName), false)
			expectFault(t, err, nil, "the entry cannot be read")
			_ = assert.ErrorAs[*fs.PathError](t, err, "the cause is the error of the file system")
		})
	})
}

// TestReadAllocs checks the allocation ceilings of the readers of a tree.
// It reads the files of a directory, which a test that runs alone measures.
func TestReadAllocs(t *testing.T) {
	alloctest.Check(t, readCases(t))
}

// BenchmarkRead measures the readers of a tree.
func BenchmarkRead(b *testing.B) {
	for _, c := range readCases(b) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// readCases returns a call of each reader of a tree, with its allocation
// ceiling, measured.
func readCases(tb testing.TB) []alloctest.Case {
	tb.Helper()
	dir := tb.TempDir()
	assert.NoError(tb, filetree.Write(dir, filetree.Tree{"a.txt": textFile}), "the tree is written")
	path := filepath.Join(dir, "a.txt")
	return []alloctest.Case{
		{Name: "Read", Call: func(assert.TB) { keptTree, errKept = filetree.Read(mapped, true) }, Allocs: 43},
		{Name: "ReadPath", Call: func(assert.TB) { _, errKept = filetree.ReadPath(path, true) }, Allocs: 8},
	}
}
