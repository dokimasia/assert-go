// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files_test

import (
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestWrite checks the tree that Write writes over a directory, and its
// faults.
func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		t.Run("creates each missing entry, replaces each file, and keeps every other entry", func(t *testing.T) {
			t.Parallel()

			dir := files.Workspace(t, files.Tree{
				"api/store.gen.go": files.Text("package api\n"),
				"go.mod":           files.Text("module example.com/a\n"),
			})
			files.Write(t, dir, files.Tree{
				"api/store.gen.go": files.Text("package api\n\nfunc Edited() {}\n"),
				"bin/run":          files.Executable("#!/bin/sh\n"),
			})
			files.Equal(t, os.DirFS(dir), files.Tree{
				"api/store.gen.go": files.Text("package api\n\nfunc Edited() {}\n"),
				"bin/run":          files.Executable("#!/bin/sh\n"),
				"go.mod":           files.Text("module example.com/a\n"),
			}, "the edited file, the new one and the kept one")
		})

		t.Run("ends with a fault for an entry of another kind, before it writes anything", func(t *testing.T) {
			t.Parallel()

			before := files.Tree{"go.mod": files.Text("module example.com/a\n")}
			dir := files.Workspace(t, before)
			seat := &matchertest.Seat{}
			files.Write(seat, dir, files.Tree{"api.txt": files.Text(""), "go.mod/a": files.Text("")})
			faults := seat.Faults()
			assert.Length(t, faults, 1, "the call ends in one fault")
			f := assert.ErrorAs[*fault.Error](t, faults[0], "a fault")
			assert.Equal(t, f.Op, "files.Write", "the operation")
			assert.Equal(t, f.Path, fault.Path{fault.Key("go.mod")}, "the file where the tree states a directory")
			assert.Equal(t, f.Reason, "the entry is a file, and the tree states a directory", "the reason")
			files.Equal(t, os.DirFS(dir), before, "Write writes nothing")
		})
	})
}

// TestWriteAllocs checks the allocation ceiling of Write.
func TestWriteAllocs(t *testing.T) {
	alloctest.Check(t, writeCases(t))
}

// BenchmarkWrite measures Write.
func BenchmarkWrite(b *testing.B) {
	for _, c := range writeCases(b) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// writeCases returns a call of Write of one short file over a file, with
// its allocation ceiling.
func writeCases(tb testing.TB) []alloctest.Case {
	tb.Helper()
	dir := files.Workspace(tb, files.Tree{"a.txt": files.Text("a\n")})
	tree := files.Tree{"a.txt": files.Text("b\n")}
	return []alloctest.Case{
		{Name: "Write", Call: func(tb assert.TB) { files.Write(tb, dir, tree) }, Allocs: 60},
	}
}
