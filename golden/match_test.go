// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The golden file of the allocation cases: its directory's mode, its name
// and its content.
const (
	goldenDirPerm = 0o755
	goldenName    = "output.txt"
	goldenContent = "recorded output"
)

// updated keeps what ShouldUpdate returns, so the compiler keeps the call.
var updated bool

// TestMatch checks MatchAt and Match against the golden files that the cases
// write, and ShouldUpdate.
func TestMatch(t *testing.T) {
	t.Parallel()

	t.Run("MatchAt", func(t *testing.T) {
		t.Parallel()

		t.Run("passes content that matches the golden file", func(t *testing.T) {
			t.Parallel()

			path := written(t, "recorded output")
			r := assert.NewRecorder()
			golden.MatchAt(r, path, []byte("recorded output"), checking)

			assert.False(t, r.Failed(), "content matching the golden file passes")
			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the call record of the pass")
		})

		t.Run("fails content that differs with a contract that names the file", func(t *testing.T) {
			t.Parallel()

			path := written(t, "recorded output")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("something else"), checking)

			records := s.Records()
			assert.Length(t, records, 1, "content differing from the golden file fails")
			assert.Contains(t, records[0].Contract, path, "the contract names the golden file")
		})

		t.Run("reports a record of golden-match-at at the caller's line for content that differs", func(t *testing.T) {
			t.Parallel()

			path := written(t, "recorded output")
			s := &matchertest.Seat{}
			_, file, line, _ := runtime.Caller(0)
			golden.MatchAt(s, path, []byte("something else"), checking)

			records := s.Records()
			assert.Length(t, records, 1, "the comparison reports one record")
			assert.Equal(t, records[0].Assertion, "golden-match-at", "the record states the comparison")
			assert.Equal(t, records[0].Detail, map[string]any{"want": "recorded output", "got": "something else"},
				"the record states the golden content as want and the output as got")
			assert.Equal(t, records[0].Where, assert.Where{File: file, Line: line + 1},
				"the record states the line that called the comparison")
		})

		t.Run("fails a missing file with a contract that states -update", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "absent.txt")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("anything"), checking)

			assert.True(t, s.Failed(), "a golden file that does not exist fails")
			assert.Contains(t, s.Records()[0].Contract, "-update", "the contract states the flag that writes the file")
		})

		t.Run("reports a record whose want is nil for a missing file", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "absent.txt")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("anything"), checking)

			records := s.Records()
			assert.Length(t, records, 1, "the comparison reports one record")
			assert.Equal(t, records[0].Detail, map[string]any{"want": nil, "got": "anything"},
				"the record states no golden content and the output")
		})

		t.Run("writes a missing file and passes while updating", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "created.txt")
			r := assert.NewRecorder()
			golden.MatchAt(r, path, []byte("fresh output"), updating)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating a missing file passes")
			assert.Equal(t, read(t, path), "fresh output", "the file contains the output")
		})

		t.Run("creates the directory of a missing file while updating", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "nested", "deeper", "created.txt")
			r := assert.NewRecorder()
			golden.MatchAt(r, path, []byte("fresh"), updating)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating into a missing directory passes")
			assert.Equal(t, read(t, path), "fresh", "the file contains the output")
		})

		t.Run("overwrites differing content and passes while updating", func(t *testing.T) {
			t.Parallel()

			path := written(t, "stale")
			r := assert.NewRecorder()
			golden.MatchAt(r, path, []byte("current"), updating)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating over stale content passes")
			assert.Equal(t, read(t, path), "current", "the file contains the new output")
		})

		t.Run("passes matching content and leaves the file while updating", func(t *testing.T) {
			t.Parallel()

			path := written(t, "at 2026-01-01T00:00:00Z exactly")
			r := assert.NewRecorder()
			golden.MatchAt(r, path, []byte("at 2026-08-30T12:00:00Z exactly"), updating, golden.ScrubTimestamps())

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating content that matches passes")
			assert.Equal(t, read(t, path), "at 2026-01-01T00:00:00Z exactly", "the file is unchanged")
		})

		t.Run("passes content that differs only in a part that a scrubber replaces", func(t *testing.T) {
			t.Parallel()

			path := written(t, "at 2026-01-01T00:00:00Z exactly")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("at 2026-08-30T12:00:00Z exactly"),
				checking, golden.ScrubTimestamps())

			assert.False(t, s.Failed(),
				"content differing only in a scrubbed timestamp passes")
		})

		t.Run("fails content that differs outside the parts that a scrubber replaces", func(t *testing.T) {
			t.Parallel()

			path := written(t, "at 2026-01-01T00:00:00Z exactly")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("at 2026-08-30T12:00:00Z roughly"),
				checking, golden.ScrubTimestamps())

			assert.True(t, s.Failed(),
				"content differing outside the scrubbed part still fails")
		})

		t.Run("ends with a fault for a golden file that cannot be read", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			golden.MatchAt(r, t.TempDir(), []byte("anything"), checking)

			assert.True(t, r.Failed(), "the fault stops the test")
			assert.Length(t, r.Failures(), 0, "the call reports no failure record")
			assert.Equal(t, verdicts(t, r), []string{"error"}, "the call record states the fault")
		})

		t.Run("ends with a fault for a directory that cannot be created while updating", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			golden.MatchAt(r, filepath.Join(dangling(t), "sub", "created.txt"), []byte("fresh"), updating)

			assert.Length(t, r.Failures(), 0, "the call reports no failure record")
			assert.Equal(t, verdicts(t, r), []string{"error"}, "the call record states the fault")
		})

		t.Run("ends with a fault for a golden file that cannot be written while updating", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			golden.MatchAt(r, dangling(t), []byte("fresh"), updating)

			assert.Length(t, r.Failures(), 0, "the call reports no failure record")
			assert.Equal(t, verdicts(t, r), []string{"error"}, "the call record states the fault")
		})
	})

	t.Run("Match", func(t *testing.T) {
		t.Parallel()

		// The case reads the path from the contract of the failure. t.Chdir
		// changes the working directory of the whole process, so a case
		// that calls it cannot run in parallel.
		t.Run("resolves the name against the conventional directory", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			golden.Match(s, "resolved.txt", []byte("output"), checking)

			records := s.Records()
			assert.Length(t, records, 1, "the golden file does not exist")
			assert.Contains(t, records[0].Contract, filepath.Join("testdata", "golden", "resolved.txt"),
				"the name resolves under testdata/golden")
		})

		t.Run("reports a record of golden-match", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			golden.Match(s, "resolved.txt", []byte("output"), checking)

			records := s.Records()
			assert.Length(t, records, 1, "the comparison reports one record")
			assert.Equal(t, records[0].Assertion, "golden-match", "the record states the comparison")
		})
	})

	t.Run("ShouldUpdate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns false unless the flag is passed", func(t *testing.T) {
			t.Parallel()

			assert.False(t, golden.ShouldUpdate(),
				"this suite is not run with -update")
		})
	})
}

// TestMatchAllocs checks the allocation ceiling of a passing call of each
// function of match.go. It changes the working directory of the process,
// which Match reads its golden file from, so it runs alone.
func TestMatchAllocs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	alloctest.Check(t, matchCases(t, dir))
}

// BenchmarkMatch measures a passing call of each function of match.go.
func BenchmarkMatch(b *testing.B) {
	dir := b.TempDir()
	b.Chdir(dir)
	for _, c := range matchCases(b, dir) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// matchCases returns a passing call of each function of match.go, with its
// allocation ceiling, measured. It writes the golden file of the calls
// under dir, the working directory of the test.
func matchCases(tb testing.TB, dir string) []alloctest.Case {
	tb.Helper()
	conventional := filepath.Join(dir, "testdata", "golden")
	assert.NoError(tb, os.MkdirAll(conventional, goldenDirPerm), "the conventional directory is made")
	path := filepath.Join(conventional, goldenName)
	assert.NoError(tb, os.WriteFile(path, []byte(goldenContent), goldenPerm), "the golden file is written")
	got := []byte(goldenContent)
	return []alloctest.Case{
		{Name: "ShouldUpdate", Call: func(assert.TB) { updated = golden.ShouldUpdate() }},
		{Name: "Match", Call: func(tb assert.TB) { golden.Match(tb, goldenName, got, checking) }, Allocs: 10},
		{Name: "MatchAt", Call: func(tb assert.TB) { golden.MatchAt(tb, path, got, checking) }, Allocs: 9},
	}
}

// written writes content to a golden file and returns its path.
func written(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "golden.txt")
	assert.NoError(t, os.WriteFile(path, []byte(content), goldenPerm),
		"the golden file for this case can be written")

	return path
}
