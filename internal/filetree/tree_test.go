// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// TestTree checks the rules of a path and of a tree, the order of its
// paths, and the directories that it implies.
func TestTree(t *testing.T) {
	t.Parallel()

	t.Run("CheckPath", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for names joined by slashes", func(t *testing.T) {
			t.Parallel()

			for _, path := range []string{"a", "a/b.txt", "ü/.hidden/x y", "..a/b.."} {
				assert.NoError(t, filetree.CheckPath(path), "the path "+path+" keeps the rules")
			}
		})

		tests := []struct {
			name       string
			give       string
			wantReason string
		}{
			{name: "returns a fault for the empty path", give: "", wantReason: `the path "" has the name ""`},
			{name: "returns a fault for an absolute path", give: "/a", wantReason: `the path "/a" has the name ""`},
			{
				name: "returns a fault for a path that ends in a slash", give: "a/",
				wantReason: `the path "a/" has the name ""`,
			},
			{
				name:       "returns a fault for a name of a dot",
				give:       "a/./b",
				wantReason: `the path "a/./b" has the name "."`,
			},
			{
				name: "returns a fault for a name of two dots", give: "a/../b",
				wantReason: `the path "a/../b" has the name ".."`,
			},
			{name: "returns a fault for a backslash", give: `a\b`, wantReason: `the path "a\\b" has the name "a\\b"`},
			{name: "returns a fault for NUL", give: "a\x00b", wantReason: `the path "a\x00b" has the name "a\x00b"`},
			{
				name: "returns a fault for a path that is no UTF-8", give: "a\xff",
				wantReason: `the path "a\xff" is no UTF-8 text`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, filetree.CheckPath(tt.give), nil, tt.wantReason)
			})
		}
	})

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a tree of every kind of entry", func(t *testing.T) {
			t.Parallel()

			tree := filetree.Tree{"a/b.txt": textFile, "a": dirWith(0o700), "run": executableFile, "l": linkTo("a")}
			assert.NoError(t, tree.Check(), "the tree keeps the rules")
		})

		tests := []struct {
			name       string
			give       filetree.Tree
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault at a path that breaks a rule of a path",
				give:       filetree.Tree{"a//b": textFile},
				wantPath:   fault.Path{fault.Key("a//b")},
				wantReason: `the path "a//b" has the name ""`,
			},
			{
				name:       "returns a fault at the zero entry",
				give:       filetree.Tree{"a": {}},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: "the entry states no file, directory or link",
			},
			{
				name:       "returns a fault at an entry of no kind",
				give:       filetree.Tree{"a": {Kind: filetree.Link + 1}},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: "the entry states no file, directory or link",
			},
			{
				name:       "returns a fault at a mode beyond the nine permission bits",
				give:       filetree.Tree{"a": fileWith("", 0o1000)},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: "the mode 0o1000 has a bit beyond the nine permission bits",
			},
			{
				name:       "returns a fault at a link that states a mode",
				give:       filetree.Tree{"a": {Kind: filetree.Link, Target: "b", Stated: true}},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: `the link to "b" states a mode, or no target of UTF-8 text without NUL`,
			},
			{
				name:       "returns a fault at a link without a target",
				give:       filetree.Tree{"a": linkTo("")},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: `the link to "" states a mode, or no target of UTF-8 text without NUL`,
			},
			{
				name:       "returns a fault at a link whose target contains NUL",
				give:       filetree.Tree{"a": linkTo("b\x00")},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: `the link to "b\x00" states a mode, or no target of UTF-8 text without NUL`,
			},
			{
				name:       "returns a fault at a link whose target is no UTF-8",
				give:       filetree.Tree{"a": linkTo("b\xff")},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: `the link to "b\xff" states a mode, or no target of UTF-8 text without NUL`,
			},
			{
				name:       "returns a fault at an entry below a file",
				give:       filetree.Tree{"a": textFile, "a/b": textFile},
				wantPath:   fault.Path{fault.Key("a/b")},
				wantReason: `the entry is below "a", which is a file`,
			},
			{
				name:       "returns a fault at an entry below a link",
				give:       filetree.Tree{"a/b/c": textFile, "a": directory, "a/b": linkTo("x")},
				wantPath:   fault.Path{fault.Key("a/b/c")},
				wantReason: `the entry is below "a/b", which is a link`,
			},
			{
				name:       "returns a fault at the first entry in path order that breaks a rule",
				give:       filetree.Tree{"b": {}, "a": {}},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: "the entry states no file, directory or link",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, tt.give.Check(), tt.wantPath, tt.wantReason)
			})
		}
	})

	t.Run("Paths", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the paths in the order of their bytes", func(t *testing.T) {
			t.Parallel()

			tree := filetree.Tree{"b": textFile, "a/b": textFile, "é": textFile, "a": directory, "a.txt": textFile}
			assert.Equal(t, tree.Paths(), []string{"a", "a.txt", "a/b", "b", "é"},
				"a dot sorts before a slash, and a parent before its entries")
		})
	})

	t.Run("Full", func(t *testing.T) {
		t.Parallel()

		t.Run("states each directory that a tree implies, without a mode", func(t *testing.T) {
			t.Parallel()

			tree := filetree.Tree{"a/b/c": textFile, "a": dirWith(0o700)}
			assert.Equal(t, tree.Full(), filetree.Tree{"a/b/c": textFile, "a": dirWith(0o700), "a/b": directory},
				"a stated directory keeps its mode")
			assert.Equal(t, tree, filetree.Tree{"a/b/c": textFile, "a": dirWith(0o700)}, "the tree is unchanged")
		})
	})
}

// TestTreeAllocs checks the allocation ceilings of the functions of a
// tree.
func TestTreeAllocs(t *testing.T) {
	alloctest.Check(t, treeCases())
}

// BenchmarkTree measures the functions of a tree.
func BenchmarkTree(b *testing.B) {
	for _, c := range treeCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// treeCases returns a call of each function of a tree, with its allocation
// ceiling, measured.
func treeCases() []alloctest.Case {
	tree := filetree.Tree{"a/b.txt": textFile, "c": directory}
	return []alloctest.Case{
		{Name: "CheckPath", Call: func(assert.TB) { errKept = filetree.CheckPath("a/b.txt") }},
		{Name: "Check", Call: func(assert.TB) { errKept = tree.Check() }, Allocs: 1},
		{Name: "Paths", Call: func(assert.TB) { keptPaths = tree.Paths() }, Allocs: 1},
		{Name: "Full", Call: func(assert.TB) { keptTree = tree.Full() }, Allocs: 2},
	}
}
