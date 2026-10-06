// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/filetree"
)

// read is a tree as a reader reads one with modes: a file, a script, a
// directory with a file, and a link.
var read = filetree.Tree{
	"a.txt":     fileWith("a\n", fileMode),
	"run.sh":    fileWith("#!/bin/sh\n", dirMode),
	"docs":      dirWith(dirMode),
	"docs/b.md": fileWith("# b\n", fileMode),
	"current":   linkTo("a.txt"),
}

// TestCompare checks the paths at which a tree read differs from a wanted
// one.
func TestCompare(t *testing.T) {
	t.Parallel()

	t.Run("Differing", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			want     filetree.Tree
			contains bool
			wantDiff []string
		}{
			{
				name: "returns no path for a tree that states each entry as read",
				want: filetree.Tree{
					"a.txt":     textFile,
					"run.sh":    executableFile,
					"docs/b.md": fileOf("# b\n"),
					"current":   linkTo("a.txt"),
				},
			},
			{
				name:     "returns a missing and an extra path in path order",
				want:     filetree.Tree{"a.txt": textFile, "b.txt": textFile, "current": linkTo("a.txt")},
				wantDiff: []string{"b.txt", "docs", "docs/b.md", "run.sh"},
			},
			{
				name:     "returns a missing path and no extra one for a tree that contains",
				want:     filetree.Tree{"a.txt": textFile, "b.txt": textFile},
				contains: true,
				wantDiff: []string{"b.txt"},
			},
			{
				name:     "returns a path whose entry is of another kind",
				want:     filetree.Tree{"docs": textFile},
				contains: true,
				wantDiff: []string{"docs"},
			},
			{
				name:     "returns a path of a file of other content",
				want:     filetree.Tree{"a.txt": fileOf("b\n")},
				contains: true,
				wantDiff: []string{"a.txt"},
			},
			{
				name:     "returns a path of a link to another target",
				want:     filetree.Tree{"current": linkTo("b.txt")},
				contains: true,
				wantDiff: []string{"current"},
			},
			{
				name:     "returns a path of a file whose stated mode differs",
				want:     filetree.Tree{"a.txt": fileWith("a\n", privateMode)},
				contains: true,
				wantDiff: []string{"a.txt"},
			},
			{
				name:     "returns a path of a directory whose stated mode differs",
				want:     filetree.Tree{"docs": dirWith(0o700)},
				contains: true,
				wantDiff: []string{"docs"},
			},
			{
				name:     "returns a path of a file whose execute bit differs where the wanted file states no mode",
				want:     filetree.Tree{"a.txt": {Kind: filetree.File, Content: "a\n", Mode: filetree.OwnerExecute}},
				contains: true,
				wantDiff: []string{"a.txt"},
			},
			{
				name:     "returns no path of a directory or a file whose mode the wanted tree does not state",
				want:     filetree.Tree{"docs/b.md": fileOf("# b\n"), "run.sh": executableFile},
				contains: true,
			},
			{
				name:     "returns a directory that the wanted tree implies, where nothing is",
				want:     filetree.Tree{"new/c.txt": textFile},
				contains: true,
				wantDiff: []string{"new", "new/c.txt"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, filetree.Differing(tt.want, read, tt.contains), tt.wantDiff, "the paths that differ")
			})
		}
	})
}

// TestCompareAllocs checks the allocation ceiling of Differing.
func TestCompareAllocs(t *testing.T) {
	alloctest.Check(t, compareCases())
}

// BenchmarkCompare measures Differing.
func BenchmarkCompare(b *testing.B) {
	for _, c := range compareCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// compareCases returns a call of Differing of a tree read with an equal
// wanted tree, with its allocation ceiling, measured.
func compareCases() []alloctest.Case {
	want := filetree.Tree{"a.txt": textFile, "docs/b.md": fileOf("# b\n")}
	got := filetree.Tree{"a.txt": textFile, "docs": directory, "docs/b.md": fileOf("# b\n")}
	return []alloctest.Case{
		{Name: "Differing", Call: func(assert.TB) { keptPaths = filetree.Differing(want, got, false) }, Allocs: 2},
	}
}
