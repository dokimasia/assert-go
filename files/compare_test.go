// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package files_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// mapped is a tree in memory: a file, a script and a link.
var mapped = fstest.MapFS{
	"a.txt":   {Data: []byte("a\n"), Mode: fileMode},
	"run.sh":  {Data: []byte("#!/bin/sh\n"), Mode: dirMode},
	"current": {Data: []byte("a.txt"), Mode: fs.ModeSymlink},
}

// mappedTree is the tree that mapped states, without its modes.
var mappedTree = files.Tree{
	"a.txt":   files.Text("a\n"),
	"run.sh":  files.Executable("#!/bin/sh\n"),
	"current": files.Link("a.txt"),
}

// TestCompare checks the comparisons of a tree read with a wanted one: the
// verdicts, the records and the faults.
func TestCompare(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		t.Run("passes a tree that has every entry and no other", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.Equal(seat, mapped, mappedTree, "the tree is built")
			assert.False(t, seat.Failed(), "the comparison passes")
		})

		t.Run("passes a directory that a workspace wrote", func(t *testing.T) {
			t.Parallel()

			tree := files.Tree{"go.mod": files.Text("module a\n"), "a/a.go": files.Text("package a\n")}
			seat := &matchertest.Seat{}
			files.Equal(seat, os.DirFS(files.Workspace(t, tree)), tree, "the tree is written")
			assert.False(t, seat.Failed(), "the directory equals the tree")
		})

		t.Run("reports a record of tree-equal at the caller's line", func(t *testing.T) {
			t.Parallel()

			want := files.Tree{"a.txt": files.Text("b\n"), "current": files.Link("a.txt")}
			seat := &matchertest.Seat{}
			_, file, line, _ := runtime.Caller(0)
			files.Equal(seat, mapped, want, "the tree is built")
			records := seat.Records()
			assert.Length(t, records, 1, "the comparison reports one record")
			assert.Equal(t, records[0], matcher.Failure{
				Assertion: "tree-equal",
				Contract:  "the tree is built",
				Detail: map[string]any{
					"want": filetree.Tree{"a.txt": {Kind: filetree.File, Content: "b\n"}},
					"got": filetree.Tree{
						"a.txt":  {Kind: filetree.File, Content: "a\n"},
						"run.sh": {Kind: filetree.File, Content: "#!/bin/sh\n", Mode: filetree.OwnerExecute},
					},
					"differences": 2,
				},
				Where: matcher.Where{File: file, Line: line + 1},
			}, "the changed file in both trees, and the extra script in got alone")
		})

		t.Run("states the mode of an entry read where the wanted entry states one", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.Equal(seat, mapped, files.Tree{
				"a.txt": files.Text("a\n").WithMode(privateMode), "run.sh": files.Executable("#!/bin/sh\n"),
				"current": files.Link("a.txt"),
			}, "the tree is built")
			assert.Equal(t, seat.Records()[0].Detail["got"], any(filetree.Tree{
				"a.txt": {Kind: filetree.File, Content: "a\n", Mode: fileMode, Stated: true},
			}), "the mode that the comparison read")
		})

		t.Run("renders the record with the sentence that the package registers", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			files.Equal(r, mapped, files.Tree{"a.txt": files.Text("a\n"), "current": files.Link("a.txt")},
				"the tree is built")
			assert.Equal(t, r.Message(), "the tree is built: 1 path differs (-want +got)"+
				"\n\trun.sh: +files.Executable(\"#!/bin/sh\\n\")", "the sentence that the package registers")
		})

		t.Run("ends with a fault for a wanted tree that breaks a rule", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.Equal(seat, mapped, files.Tree{"a.txt": {}}, "the tree is built")
			expectFault(
				t,
				seat,
				"files.Equal",
				fault.Path{fault.Key("a.txt")},
				"the entry states no file, directory or link",
			)
		})

		t.Run("ends with a fault for a tree that cannot be read", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.Equal(seat, fstest.MapFS{"pipe": {Mode: fs.ModeNamedPipe}}, files.Tree{}, "the tree is built")
			expectFault(
				t,
				seat,
				"files.Equal",
				fault.Path{fault.Key("pipe")},
				"the entry is no file, directory or link",
			)
		})
	})

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()

		t.Run("passes a tree that has every wanted entry and more", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.Contains(seat, mapped, files.Tree{"a.txt": files.Text("a\n")}, "the tree has the file")
			assert.False(t, seat.Failed(), "the extra entries are no difference")
		})

		t.Run("reports a record of tree-contains for a missing entry", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.Contains(seat, mapped, files.Tree{"b.txt": files.Text("b\n")}, "the tree has the file")
			records := seat.Records()
			assert.Length(t, records, 1, "the comparison reports one record")
			assert.Equal(t, records[0].Assertion, "tree-contains", "the assertion")
			assert.Equal(t, records[0].Detail, map[string]any{
				"want":        filetree.Tree{"b.txt": {Kind: filetree.File, Content: "b\n"}},
				"got":         filetree.Tree{},
				"differences": 1,
			}, "the missing entry in want alone")
		})

		t.Run("ends with a fault for a wanted tree that breaks a rule", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.Contains(seat, mapped, files.Tree{"a.txt/b": files.Text(""), "a.txt": files.Text("")}, "the tree")
			expectFault(
				t,
				seat,
				"files.Contains",
				fault.Path{fault.Key("a.txt/b")},
				`the entry is below "a.txt", which is a file`,
			)
		})
	})

	t.Run("Unchanged", func(t *testing.T) {
		t.Parallel()

		t.Run("passes a call that reads and rewrites every file", func(t *testing.T) {
			t.Parallel()

			dir := files.Workspace(
				t,
				files.Tree{"a.txt": files.Text("a\n"), "keys/id": files.Text("k").WithMode(privateMode)},
			)
			seat := &matchertest.Seat{}
			files.Unchanged(seat, os.DirFS(dir), func() {
				for _, name := range []string{"a.txt", "keys/id"} {
					path := filepath.Join(dir, name)
					content, err := os.ReadFile(path)
					assert.NoError(t, err, "the file is read")
					assert.NoError(t, os.WriteFile(path, content, fileMode), "the file is written again")
				}
			}, "a rewrite changes nothing")
			assert.False(t, seat.Failed(), "the bytes and the modes are unchanged")
		})

		t.Run("reports a record of tree-unchanged for a call that writes a file", func(t *testing.T) {
			t.Parallel()

			dir := files.Workspace(t, files.Tree{"a.txt": files.Text("a\n")})
			seat := &matchertest.Seat{}
			files.Unchanged(seat, os.DirFS(dir), func() {
				assert.NoError(
					t,
					os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), privateMode),
					"the call writes",
				)
			}, "a dry run writes nothing")
			records := seat.Records()
			assert.Length(t, records, 1, "the comparison reports one record")
			assert.Equal(t, records[0].Assertion, "tree-unchanged", "the assertion")
			assert.Equal(t, records[0].Detail, map[string]any{
				"want":        filetree.Tree{},
				"got":         filetree.Tree{"new.txt": {Kind: filetree.File, Content: "new"}},
				"differences": 1,
			}, "the new file in got alone, without its mode")
		})

		t.Run("reports a change of a mode", func(t *testing.T) {
			t.Parallel()

			dir := files.Workspace(t, files.Tree{"a.txt": files.Text("a\n")})
			seat := &matchertest.Seat{}
			files.Unchanged(seat, os.DirFS(dir), func() {
				assert.NoError(t, os.Chmod(filepath.Join(dir, "a.txt"), privateMode), "the call changes the mode")
			}, "a dry run writes nothing")
			assert.Equal(t, seat.Records()[0].Detail["differences"], any(1), "the mode is part of the tree")
		})

		t.Run("ends with the panic of the call", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			got := assert.Panics(t, func() {
				files.Unchanged(seat, mapped, func() { panic("the call fails") }, "a dry run writes nothing")
			}, "the panic reaches the caller")
			assert.Equal(t, got, any("the call fails"), "the call's own panic")
			assert.False(t, seat.Failed(), "the comparison reports nothing")
		})

		t.Run("ends with a fault for a tree that cannot be read after the call", func(t *testing.T) {
			t.Parallel()

			dir := files.Workspace(t, files.Tree{"a.txt": files.Text("a\n")})
			seat := &matchertest.Seat{}
			files.Unchanged(seat, os.DirFS(dir), func() {
				assert.NoError(t, os.RemoveAll(dir), "the call removes the directory")
			}, "a dry run writes nothing")
			expectFault(t, seat, "files.Unchanged", nil, "the tree cannot be read")
		})
	})
}

// expectFault checks that the call on seat ended in one fault of op, at path,
// whose reason is reason.
func expectFault(t *testing.T, seat *matchertest.Seat, op string, path fault.Path, reason string) {
	t.Helper()

	faults := seat.Faults()
	assert.Length(t, faults, 1, "the call ends in one fault")
	f := assert.ErrorAs[*fault.Error](t, faults[0], "a fault")
	assert.Equal(t, f.Op, op, "the operation")
	assert.Equal(t, f.Path, path, "the path of the fault")
	assert.Equal(t, f.Reason, reason, "the reason")
}

// TestCompareAllocs checks the allocation ceilings of the comparisons of
// trees.
func TestCompareAllocs(t *testing.T) {
	alloctest.Check(t, compareCases())
}

// BenchmarkCompare measures the comparisons of trees.
func BenchmarkCompare(b *testing.B) {
	for _, c := range compareCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// compareCases returns a passing call of each comparison of the tree in
// memory, with its allocation ceiling, measured.
func compareCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "Equal", Call: func(tb assert.TB) { files.Equal(tb, mapped, mappedTree, "the tree") }, Allocs: 43},
		{Name: "Contains", Call: func(tb assert.TB) { files.Contains(tb, mapped, mappedTree, "the tree") }, Allocs: 43},
		{
			Name:   "Unchanged",
			Call:   func(tb assert.TB) { files.Unchanged(tb, mapped, func() {}, "the tree") },
			Allocs: 78,
		},
	}
}
