// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/matchertest"
)

// Whether a run may rewrite its golden files. Named rather than
// repeated, because a case passing the wrong one silently tests the
// other path.
const (
	updating = true
	checking = false
)

// goldenPerm is what a golden file a test writes is created with.
const goldenPerm = 0o644

func TestMatch(t *testing.T) {
	t.Parallel()

	t.Run("MatchAt", func(t *testing.T) {
		t.Parallel()

		t.Run("matching content reports nothing", func(t *testing.T) {
			t.Parallel()

			path := written(t, "recorded output")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("recorded output"), checking)

			assert.False(t, s.Failed(), "content matching the golden file passes")
		})

		t.Run("differing content reports a diff naming the file", func(t *testing.T) {
			t.Parallel()

			path := written(t, "recorded output")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("something else"), checking)

			assert.True(t, s.Failed(), "content differing from the golden file fails")
			assert.ContainsInOrder(t, s.First(), []string{path, "-want +got"},
				"the failure names the file, then shows the diff")
		})

		t.Run("a missing file names the flag that would create it", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "absent.txt")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("anything"), checking)

			assert.True(t, s.Failed(), "a golden file that does not exist fails")
			assert.Contains(t, s.First(), "-update",
				"the failure names the flag that would create it")
		})

		t.Run("updating writes a missing file and passes", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "created.txt")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("fresh output"), updating)

			assert.False(t, s.Failed(), "updating a missing file passes")
			assert.Equal(t, read(t, path), "fresh output",
				"the file holds the output it was given")
		})

		t.Run("updating creates the directory", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "nested", "deeper", "created.txt")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("fresh"), updating)

			assert.False(t, s.Failed(), "updating into a missing directory passes")
		})

		t.Run("updating overwrites differing content", func(t *testing.T) {
			t.Parallel()

			path := written(t, "stale")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("current"), updating)

			assert.False(t, s.Failed(), "updating over stale content passes")
			assert.Equal(t, read(t, path), "current", "the file holds the new output")
		})

		t.Run("a scrubber is applied to both sides", func(t *testing.T) {
			t.Parallel()

			path := written(t, "at 2026-01-01T00:00:00Z exactly")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("at 2026-08-30T12:00:00Z exactly"),
				checking, golden.ScrubTimestamps())

			assert.False(t, s.Failed(),
				"content differing only in a scrubbed timestamp passes")
		})

		t.Run("a scrubber does not hide a real difference", func(t *testing.T) {
			t.Parallel()

			path := written(t, "at 2026-01-01T00:00:00Z exactly")
			s := &matchertest.Seat{}
			golden.MatchAt(s, path, []byte("at 2026-08-30T12:00:00Z roughly"),
				checking, golden.ScrubTimestamps())

			assert.True(t, s.Failed(),
				"content differing outside the scrubbed part still fails")
		})
	})

	t.Run("Match", func(t *testing.T) {
		t.Parallel()

		// The path is read out of the failure rather than by changing
		// the working directory. t.Chdir is process-global and cannot
		// run beside a parallel sibling, and this establishes the same
		// thing without touching anything outside the test.
		t.Run("resolves against the conventional directory", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			golden.Match(s, "resolved.txt", []byte("output"), checking)

			assert.Contains(t, s.First(), filepath.Join("testdata", "golden", "resolved.txt"),
				"the name resolves under testdata/golden")
		})
	})

	t.Run("ShouldUpdate", func(t *testing.T) {
		t.Parallel()

		t.Run("is false unless the flag is passed", func(t *testing.T) {
			t.Parallel()

			assert.False(t, golden.ShouldUpdate(),
				"this suite is not run with -update")
		})
	})
}

// written puts content in a golden file and answers its path.
func written(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "golden.txt")
	assert.NoError(t, os.WriteFile(path, []byte(content), goldenPerm),
		"the golden file for this case can be written")

	return path
}

// read answers what a golden file holds.
func read(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	assert.NoError(t, err, "the golden file can be read back")

	return string(raw)
}
