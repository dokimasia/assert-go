// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package files_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// pathTree is the workspace of the tests of one path: a file of text, a file
// of bytes, a private key, a directory and two links.
var pathTree = files.Tree{
	"a.txt":    files.Text("a\n"),
	"logo.png": files.Bytes([]byte{0x89, 'P', 'N', 'G'}),
	"keys":     files.Dir().WithMode(0o700),
	"keys/id":  files.Text("secret\n").WithMode(privateMode),
	"current":  files.Link("a.txt"),
	"dangling": files.Link("missing.txt"),
}

// TestPath checks the assertions of one path: their verdicts, their records
// and their faults.
func TestPath(t *testing.T) {
	t.Parallel()

	dir := files.Workspace(t, pathTree)
	at := func(name string) string { return filepath.Join(dir, name) }

	tests := []struct {
		name       string
		call       func(tb assert.TB)
		modes      bool
		wantID     string
		wantDetail map[string]any
	}{
		{name: "Absent passes where nothing is", call: func(tb assert.TB) { files.Absent(tb, at("b.txt"), "gone") }},
		{name: "Absent passes below a file", call: func(tb assert.TB) { files.Absent(tb, at("a.txt/b"), "gone") }},
		{
			name:   "Absent fails at a link whose target is missing",
			call:   func(tb assert.TB) { files.Absent(tb, at("dangling"), "gone") },
			wantID: "path-absent", wantDetail: map[string]any{"got": "link"},
		},
		{name: "IsFile passes at a file", call: func(tb assert.TB) { files.IsFile(tb, at("a.txt"), "a file") }},
		{
			name:   "IsFile fails at a link to a file",
			call:   func(tb assert.TB) { files.IsFile(tb, at("current"), "a file") },
			wantID: "is-file", wantDetail: map[string]any{"got": "link"},
		},
		{
			name:   "IsFile fails where nothing is",
			call:   func(tb assert.TB) { files.IsFile(tb, at("b.txt"), "a file") },
			wantID: "is-file", wantDetail: map[string]any{"got": nil},
		},
		{name: "IsDir passes at a directory", call: func(tb assert.TB) { files.IsDir(tb, at("keys"), "a directory") }},
		{
			name:   "IsDir fails at a file",
			call:   func(tb assert.TB) { files.IsDir(tb, at("a.txt"), "a directory") },
			wantID: "is-dir", wantDetail: map[string]any{"got": "file"},
		},
		{
			name: "LinksTo passes at a link to the target",
			call: func(tb assert.TB) { files.LinksTo(tb, at("current"), "a.txt", "a link") },
		},
		{
			name:   "LinksTo fails at a link to another target",
			call:   func(tb assert.TB) { files.LinksTo(tb, at("current"), "b.txt", "a link") },
			wantID: "links-to", wantDetail: map[string]any{"want": "b.txt", "got": "a.txt", "kind": "link"},
		},
		{
			name:   "LinksTo fails at a file",
			call:   func(tb assert.TB) { files.LinksTo(tb, at("a.txt"), "b.txt", "a link") },
			wantID: "links-to", wantDetail: map[string]any{"want": "b.txt", "got": nil, "kind": "file"},
		},
		{
			name: "HasContent passes at a file of the content",
			call: func(tb assert.TB) { files.HasContent(tb, at("a.txt"), "a\n", "the content") },
		},
		{
			name:   "HasContent fails at a file of other text",
			call:   func(tb assert.TB) { files.HasContent(tb, at("a.txt"), "a\r\n", "the content") },
			wantID: "has-content", wantDetail: map[string]any{"want": "a\r\n", "got": "a\n", "kind": "file"},
		},
		{
			name:   "HasContent states content that is no UTF-8 as bytes",
			call:   func(tb assert.TB) { files.HasContent(tb, at("logo.png"), "\x89PNH", "the content") },
			wantID: "has-content", wantDetail: map[string]any{
				"want": []byte{0x89, 'P', 'N', 'H'}, "got": []byte{0x89, 'P', 'N', 'G'}, "kind": "file",
			},
		},
		{
			name:   "HasContent fails at a directory",
			call:   func(tb assert.TB) { files.HasContent(tb, at("keys"), "", "the content") },
			wantID: "has-content", wantDetail: map[string]any{"want": "", "got": nil, "kind": "directory"},
		},
		{
			name: "HasMode passes at a file of the mode", modes: true,
			call: func(tb assert.TB) { files.HasMode(tb, at("keys/id"), privateMode, "private") },
		},
		{
			name: "HasMode passes at a directory of the mode", modes: true,
			call: func(tb assert.TB) { files.HasMode(tb, at("keys"), 0o700, "private") },
		},
		{
			name: "HasMode fails at a file of another mode", modes: true,
			call:   func(tb assert.TB) { files.HasMode(tb, at("a.txt"), privateMode, "private") },
			wantID: "has-mode", wantDetail: map[string]any{"want": privateMode, "got": fileMode, "kind": "file"},
		},
		{
			name: "HasMode fails at a link, which has no mode", modes: true,
			call:   func(tb assert.TB) { files.HasMode(tb, at("current"), fileMode, "private") },
			wantID: "has-mode", wantDetail: map[string]any{"want": fileMode, "got": nil, "kind": "link"},
		},
		{
			name: "HasMode fails where nothing is", modes: true,
			call:   func(tb assert.TB) { files.HasMode(tb, at("b.txt"), fileMode, "private") },
			wantID: "has-mode", wantDetail: map[string]any{"want": fileMode, "got": nil, "kind": nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.modes {
				mustRecordModes(t)
			}
			seat := &matchertest.Seat{}
			tt.call(seat)
			records := seat.Records()
			if tt.wantDetail == nil {
				assert.Empty(t, records, "the assertion passes")
				return
			}
			assert.Length(t, records, 1, "the assertion reports one record")
			assert.Equal(t, records[0].Assertion, tt.wantID, "the assertion")
			assert.Equal(t, records[0].Detail, tt.wantDetail, "the detail of the record")
		})
	}

	t.Run("Absent", func(t *testing.T) {
		t.Parallel()

		t.Run("ends with a fault for an entry that is no file, directory or link", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.Absent(seat, os.DevNull, "gone")
			expectFault(t, seat, "files.Absent", nil, "the entry is no file, directory or link")
		})
	})

	t.Run("IsFile", func(t *testing.T) {
		t.Parallel()

		t.Run("ends with a fault for a path that cannot be read", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.IsFile(seat, at(longName), "a file")
			expectFault(t, seat, "files.IsFile", nil, "the entry cannot be read")
		})
	})

	t.Run("HasMode", func(t *testing.T) {
		t.Parallel()

		t.Run("ends with a fault for a mode beyond the nine permission bits", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			files.HasMode(seat, at("a.txt"), fs.ModeDir|fileMode, "private")
			expectFault(
				t,
				seat,
				"files.HasMode",
				nil,
				"the mode 0o20000000644 has a bit beyond the nine permission bits",
			)
		})
	})
}

// TestPathAllocs checks the allocation ceilings of the assertions of one
// path. They read the entries of a directory, which a test that runs alone
// measures.
func TestPathAllocs(t *testing.T) {
	alloctest.Check(t, pathCases(t))
}

// BenchmarkPath measures the assertions of one path.
func BenchmarkPath(b *testing.B) {
	for _, c := range pathCases(b) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// pathCases returns a passing call of each assertion of one path, with its
// allocation ceiling, measured. HasMode passes only where the file systems
// record permission bits.
func pathCases(tb testing.TB) []alloctest.Case {
	tb.Helper()
	mustRecordModes(tb)
	dir := files.Workspace(tb, pathTree)
	at := func(name string) string { return filepath.Join(dir, name) }
	absent, file, keys, current := at("b.txt"), at("a.txt"), at("keys"), at("current")
	return []alloctest.Case{
		{Name: "Absent", Call: func(tb assert.TB) { files.Absent(tb, absent, "gone") }, Allocs: 3},
		{Name: "IsFile", Call: func(tb assert.TB) { files.IsFile(tb, file, "a file") }, Allocs: 2},
		{Name: "IsDir", Call: func(tb assert.TB) { files.IsDir(tb, keys, "a directory") }, Allocs: 2},
		{Name: "LinksTo", Call: func(tb assert.TB) { files.LinksTo(tb, current, "a.txt", "a link") }, Allocs: 5},
		{Name: "HasContent", Call: func(tb assert.TB) { files.HasContent(tb, file, "a\n", "the content") }, Allocs: 8},
		{Name: "HasMode", Call: func(tb assert.TB) { files.HasMode(tb, keys, 0o700, "private") }, Allocs: 2},
	}
}
