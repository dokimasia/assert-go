// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// apiDir is the golden directory of the name api, relative to the test's
// working directory.
var apiDir = filepath.Join("testdata", "golden", "api")

// api is the output of the cases: a file, a script and a file in a
// directory, as a tree in memory.
var api = fstest.MapFS{
	"api.go":      {Data: []byte("package api\n"), Mode: 0o644},
	"gen.sh":      {Data: []byte("#!/bin/sh\n"), Mode: 0o755},
	"v1/types.go": {Data: []byte("package v1\n"), Mode: 0o644},
}

// apiTree is the tree of api without its modes, as a golden tree states it.
var apiTree = filetree.Tree{
	"api.go":      {Kind: filetree.File, Content: "package api\n"},
	"gen.sh":      {Kind: filetree.File, Content: "#!/bin/sh\n", Mode: filetree.OwnerExecute},
	"v1":          {Kind: filetree.Dir},
	"v1/types.go": {Kind: filetree.File, Content: "package v1\n"},
}

// TestTree checks MatchTree against the golden trees that the cases write.
// Each case changes the working directory of the process to a temporary
// directory of its own, which MatchTree resolves the conventional directory
// against, so the cases run one at a time.
func TestTree(t *testing.T) {
	t.Run("MatchTree", func(t *testing.T) {
		t.Run("passes an output that equals the golden tree", func(t *testing.T) {
			t.Chdir(t.TempDir())
			goldenTree(t, apiTree)
			r := assert.NewRecorder()
			golden.MatchTree(r, "api", api, checking)
			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the call record of the pass")
		})

		t.Run("passes a golden tree whose modes differ in no execute bit", func(t *testing.T) {
			t.Chdir(t.TempDir())
			goldenTree(t, filetree.Tree{
				"api.go":      {Kind: filetree.File, Content: "package api\n", Mode: 0o600, Stated: true},
				"gen.sh":      {Kind: filetree.File, Content: "#!/bin/sh\n", Mode: 0o700, Stated: true},
				"v1":          {Kind: filetree.Dir, Mode: 0o700, Stated: true},
				"v1/types.go": {Kind: filetree.File, Content: "package v1\n"},
			})
			s := &matchertest.Seat{}
			golden.MatchTree(s, "api", api, checking)
			assert.False(t, s.Failed(), "the comparison reads no mode of the golden tree")
		})

		t.Run("reports a record of golden-match-tree at the caller's line for a golden tree that differs",
			func(t *testing.T) {
				mustRecordModes(t)
				t.Chdir(t.TempDir())
				goldenTree(t, filetree.Tree{
					"api.go":      {Kind: filetree.File, Content: "package api\n\nfunc Old() {}\n"},
					"gen.sh":      {Kind: filetree.File, Content: "#!/bin/sh\n"},
					"v1/types.go": {Kind: filetree.File, Content: "package v1\n"},
				})
				s := &matchertest.Seat{}
				_, file, line, _ := runtime.Caller(0)
				golden.MatchTree(s, "api", api, checking)
				records := s.Records()
				assert.Length(t, records, 1, "the comparison reports one record")
				assert.Equal(t, records[0], matcher.Failure{
					Assertion: "golden-match-tree",
					Contract:  "the golden tree " + apiDir + " matches the output, and -update writes it",
					Detail: map[string]any{
						"want": filetree.Tree{
							"api.go": {Kind: filetree.File, Content: "package api\n\nfunc Old() {}\n"},
							"gen.sh": {Kind: filetree.File, Content: "#!/bin/sh\n"},
						},
						"got":         filetree.Tree{"api.go": apiTree["api.go"], "gen.sh": apiTree["gen.sh"]},
						"differences": 2,
					},
					Where: matcher.Where{File: file, Line: line + 1},
				}, "the changed file and the execute bit that differs")
			})

		t.Run("fails a missing golden tree with want nil and the output's entries", func(t *testing.T) {
			mustRecordModes(t)
			t.Chdir(t.TempDir())
			s := &matchertest.Seat{}
			golden.MatchTree(s, "api", api, checking)
			records := s.Records()
			assert.Length(t, records, 1, "the comparison reports one record")
			assert.Equal(t, records[0].Detail, map[string]any{"want": nil, "got": apiTree, "differences": 4},
				"no golden tree, and the output's four entries")
			assert.Contains(t, records[0].Contract, "-update", "the contract states the flag that writes the tree")
		})

		t.Run("renders the record with the sentence of a comparison of trees", func(t *testing.T) {
			t.Chdir(t.TempDir())
			r := assert.NewRecorder()
			golden.MatchTree(r, "api", fstest.MapFS{"a.txt": {Data: []byte("a\n")}}, checking)
			assert.Equal(t, r.Message(), "the golden tree "+apiDir+" matches the output, and -update writes it: "+
				"the golden tree is missing, and the output has 1 entries (+got)\n\ta.txt: +files.Text(\"a\\n\")",
				"the sentence that the package registers")
		})

		t.Run("writes a missing golden tree and passes while updating", func(t *testing.T) {
			mustRecordModes(t)
			t.Chdir(t.TempDir())
			r := assert.NewRecorder()
			golden.MatchTree(r, "api", api, updating)
			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the update passes")
			assert.Equal(t, readTree(t), apiTree, "the golden tree is the output")
		})

		t.Run("rewrites a golden tree that differs without its extra entries while updating", func(t *testing.T) {
			mustRecordModes(t)
			t.Chdir(t.TempDir())
			outside := filepath.Join(t.TempDir(), "keep.txt")
			assert.NoError(t, os.WriteFile(outside, []byte("kept"), goldenPerm), "a file outside is written")
			goldenTree(t, filetree.Tree{
				"api.go":       {Kind: filetree.File, Content: "package old\n"},
				"latest":       {Kind: filetree.Link, Target: outside},
				"old/stale.go": {Kind: filetree.File, Content: "package old\n"},
			})
			r := assert.NewRecorder()
			golden.MatchTree(r, "api", api, updating)
			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the update passes")
			assert.Equal(t, readTree(t), apiTree, "the golden tree is the output")
			assert.Equal(t, read(t, outside), "kept", "the entry that the removed link points to stays")
		})

		t.Run("leaves an equal golden tree as it is while updating", func(t *testing.T) {
			mustRecordModes(t)
			t.Chdir(t.TempDir())
			goldenTree(
				t,
				filetree.Tree{"api.go": {Kind: filetree.File, Content: "package api\n", Mode: 0o600, Stated: true}},
			)
			r := assert.NewRecorder()
			golden.MatchTree(r, "api", fstest.MapFS{"api.go": {Data: []byte("package api\n")}}, updating)
			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the update passes")
			info, err := os.Lstat(filepath.Join(apiDir, "api.go"))
			assert.NoError(t, err, "the golden file is there")
			assert.Equal(t, info.Mode().Perm(), fs.FileMode(0o600), "the golden file keeps its mode")
		})

		t.Run("compares and writes the content that the scrubbers leave", func(t *testing.T) {
			t.Chdir(t.TempDir())
			output := fstest.MapFS{"log.txt": {Data: []byte("at 2026-08-30T12:00:00Z\n")}}
			r := assert.NewRecorder()
			golden.MatchTree(r, "api", output, updating, golden.ScrubTimestamps())
			assert.Equal(t, read(t, filepath.Join(apiDir, "log.txt")), "at SCRUBBED_TIMESTAMP\n",
				"the update writes the scrubbed content")
			s := &matchertest.Seat{}
			later := fstest.MapFS{"log.txt": {Data: []byte("at 2026-10-06T09:00:00Z\n")}}
			golden.MatchTree(s, "api", later, checking, golden.ScrubTimestamps())
			assert.False(t, s.Failed(), "an output that differs only in a timestamp passes")
		})

		faults := []struct {
			name       string
			give       string
			output     fs.FS
			update     bool
			setup      func(t *testing.T)
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "ends with a fault for a name that is no path", give: "../api", output: api,
				wantReason: `the path "../api" has the name ".."`,
			},
			{
				name: "ends with a fault for an output that cannot be read", give: "api",
				output: fstest.MapFS{"pipe": {Mode: fs.ModeNamedPipe}}, wantPath: fault.Path{fault.Key("pipe")},
				wantReason: "the entry is no file, directory or link",
			},
			{
				name: "ends with a fault for a golden tree that cannot be read", give: "api", output: api,
				setup: func(t *testing.T) {
					t.Helper()
					if runtime.GOOS == "windows" {
						t.Skip(
							"Windows opens the root of a file system on a file as the file, which reads as no directory",
						)
					}
					assert.NoError(t, os.MkdirAll(filepath.Dir(apiDir), 0o755), "the conventional directory is made")
					assert.NoError(t, os.WriteFile(apiDir, []byte("a file"), goldenPerm), "a file is the golden tree")
				},
				wantReason: "the tree cannot be read",
			},
			{
				name: "ends with a fault for a golden tree that cannot be written while updating", give: "api",
				output: api, update: updating,
				setup: func(t *testing.T) {
					t.Helper()
					assert.NoError(t, os.Symlink(filepath.Join(t.TempDir(), "missing"), "testdata"),
						"the test data directory is a link whose target is missing")
				},
				wantReason: "the directory cannot be created",
			},
		}
		for _, tt := range faults {
			t.Run(tt.name, func(t *testing.T) {
				t.Chdir(t.TempDir())
				if tt.setup != nil {
					tt.setup(t)
				}
				s := &matchertest.Seat{}
				golden.MatchTree(s, tt.give, tt.output, tt.update)
				faults := s.Faults()
				assert.Length(t, faults, 1, "the call ends in one fault")
				f := assert.ErrorAs[*fault.Error](t, faults[0], "a fault")
				assert.Equal(t, f.Op, "golden.MatchTree", "the operation")
				assert.Equal(t, f.Path, tt.wantPath, "the path of the fault")
				assert.Equal(t, f.Reason, tt.wantReason, "the reason")
			})
		}
	})
}

// goldenTree writes tree as the golden tree of the name api, under the
// working directory.
func goldenTree(t *testing.T, tree filetree.Tree) {
	t.Helper()
	assert.NoError(t, os.MkdirAll(apiDir, 0o755), "the golden directory is made")
	assert.NoError(t, filetree.Write(apiDir, tree), "the golden tree is written")
}

// readTree returns the golden tree of the name api, without its modes.
func readTree(t *testing.T) filetree.Tree {
	t.Helper()
	tree, err := filetree.Read(os.DirFS(apiDir), false)
	assert.NoError(t, err, "the golden tree is read")
	return tree
}

// TestTreeAllocs checks the allocation ceiling of a passing call of
// MatchTree. It changes the working directory of the process, which
// MatchTree reads its golden tree from, so it runs alone.
func TestTreeAllocs(t *testing.T) {
	t.Chdir(t.TempDir())
	alloctest.Check(t, treeCases(t))
}

// BenchmarkTree measures a passing call of MatchTree.
func BenchmarkTree(b *testing.B) {
	b.Chdir(b.TempDir())
	for _, c := range treeCases(b) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// treeCases returns a passing call of MatchTree of the output api against
// its golden tree, which it writes under the working directory, with its
// allocation ceiling, measured.
func treeCases(tb testing.TB) []alloctest.Case {
	tb.Helper()
	assert.NoError(tb, os.MkdirAll(apiDir, 0o755), "the golden directory is made")
	assert.NoError(tb, filetree.Write(apiDir, apiTree), "the golden tree is written")
	return []alloctest.Case{
		{Name: "MatchTree", Call: func(tb assert.TB) { golden.MatchTree(tb, "api", api, checking) }, Allocs: 122},
	}
}
