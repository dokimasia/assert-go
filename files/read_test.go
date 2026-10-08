// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package files_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestRead checks the content that Read returns, and its faults.
func TestRead(t *testing.T) {
	t.Parallel()

	dir := files.Workspace(t, files.Tree{
		"a.txt":   files.Text("version 2\n"),
		"docs":    files.Dir(),
		"current": files.Link("a.txt"),
	})

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the content of the file at the path", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, files.Read(t, filepath.Join(dir, "a.txt")), "version 2\n", "the file's content")
		})

		tests := []struct {
			name       string
			path       string
			wantReason string
		}{
			{
				name: "ends with a fault where nothing is", path: "b.txt",
				wantReason: fmt.Sprintf("no file is at the path %q", filepath.Join(dir, "b.txt")),
			},
			{
				name: "ends with a fault at a directory", path: "docs",
				wantReason: fmt.Sprintf("no file is at the path %q", filepath.Join(dir, "docs")),
			},
			{
				name: "ends with a fault at a link to a file", path: "current",
				wantReason: fmt.Sprintf("no file is at the path %q", filepath.Join(dir, "current")),
			},
			{
				name: "ends with a fault for a path that cannot be read", path: longName,
				wantReason: "the entry cannot be read",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				seat := &matchertest.Seat{}
				got := files.Read(seat, filepath.Join(dir, tt.path))
				assert.Equal(t, got, "", "no content")
				faults := seat.Faults()
				assert.Length(t, faults, 1, "the call ends in one fault")
				f := assert.ErrorAs[*fault.Error](t, faults[0], "a fault")
				assert.Equal(t, f.Op, "files.Read", "the operation")
				assert.Equal(t, f.Reason, tt.wantReason, "the reason")
			})
		}
	})
}

// TestReadAllocs checks the allocation ceiling of Read.
func TestReadAllocs(t *testing.T) {
	alloctest.Check(t, readCases(t))
}

// BenchmarkRead measures Read.
func BenchmarkRead(b *testing.B) {
	for _, c := range readCases(b) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// readCases returns a call of Read of a short file, with its allocation
// ceiling, measured.
func readCases(tb testing.TB) []alloctest.Case {
	tb.Helper()
	path := filepath.Join(files.Workspace(tb, files.Tree{"a.txt": files.Text("a\n")}), "a.txt")
	return []alloctest.Case{
		{Name: "Read", Call: func(tb assert.TB) { kept = files.Read(tb, path) }, Allocs: 8},
	}
}
