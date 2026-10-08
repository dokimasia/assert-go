// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package filetree_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/filetree"
)

// TestEntry checks the kinds of an entry and its text.
func TestEntry(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the kind as a record states it", func(t *testing.T) {
			t.Parallel()

			names := []string{filetree.None.Name(), filetree.File.Name(), filetree.Dir.Name(), filetree.Link.Name()}
			assert.Equal(t, names, []string{"", "file", "directory", "link"}, "the names of the four kinds")
		})
	})

	t.Run("Executable", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give filetree.Entry
			want bool
		}{
			{name: "reports true for a file with the owner's execute bit", give: executableFile, want: true},
			{name: "reports false for a file without it", give: textFile, want: false},
			{name: "reports false for a directory with the bit", give: dirWith(0o755), want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Executable(), tt.want, "whether the owner may execute the entry")
			})
		}
	})

	t.Run("GoString", func(t *testing.T) {
		t.Parallel()

		large := strings.Repeat("a", filetree.ContentLimit+1)
		tests := []struct {
			name string
			give filetree.Entry
			want string
		}{
			{name: "returns Text of a file", give: textFile, want: `files.Text("a\n")`},
			{
				name: "returns Executable of a file that its owner may execute", give: executableFile,
				want: `files.Executable("#!/bin/sh\n")`,
			},
			{
				name: "returns Text and WithMode of a file that states its mode", give: fileWith("k", 0o700),
				want: `files.Text("k").WithMode(0o700)`,
			},
			{name: "quotes bytes that are no UTF-8", give: fileOf("\x89PNG"), want: `files.Text("\x89PNG")`},
			{name: "returns Dir of a directory", give: filetree.Entry{Kind: filetree.Dir}, want: "files.Dir()"},
			{
				name: "returns Dir and WithMode of a directory that states its mode", give: dirWith(0o750),
				want: "files.Dir().WithMode(0o750)",
			},
			{name: "returns Link of a link", give: linkTo("a.txt"), want: `files.Link("a.txt")`},
			{name: "returns none for the zero entry", give: filetree.Entry{}, want: "none"},
			{
				name: "states the size and the digest of a content over the limit",
				give: fileOf(large),
				want: "files.Text(<65537 bytes, sha256:008ffc88d3c96a9f307524eb361e47c5222a887fc45fa0c1fb8d429c5c23b430>)",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.GoString(), tt.want, "the Go expression of the entry")
			})
		}
	})
}

// TestEntryAllocs checks the allocation ceilings of the functions of an
// entry.
func TestEntryAllocs(t *testing.T) {
	alloctest.Check(t, entryCases())
}

// BenchmarkEntry measures the functions of an entry.
func BenchmarkEntry(b *testing.B) {
	for _, c := range entryCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// entryCases returns a call of each function of an entry, with its
// allocation ceiling, measured.
func entryCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "Name", Call: func(assert.TB) { kept = filetree.File.Name() }},
		{Name: "Executable", Call: func(assert.TB) { keptBool = executableFile.Executable() }},
		{Name: "GoString", Call: func(assert.TB) { kept = textFile.GoString() }, Allocs: 2},
	}
}
