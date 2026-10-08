// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/files"
)

// TestSubject checks the table of built behaviours against the
// definition's subjects table, and the shapes of a behaviour that no
// corpus case and no vector observes.
func TestSubject(t *testing.T) {
	t.Parallel()

	t.Run("Subjects", func(t *testing.T) {
		t.Parallel()

		t.Run("builds every kind of the definition's subjects table and no other", func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile("spec/assertions.json")
			assert.NoError(t, err, "the vendored assertion table can be read")
			var doc struct {
				Subjects map[string]json.RawMessage `json:"subjects"`
			}
			assert.NoError(t, json.Unmarshal(raw, &doc), "the vendored assertion table parses")
			assert.NotEmpty(t, doc.Subjects, "the definition states subjects")
			assert.Equal(t, slices.Sorted(maps.Keys(conformance.Subjects)), slices.Sorted(maps.Keys(doc.Subjects)),
				"the kinds that the table builds")
		})

		t.Run("builds a subject with state of its own on each call", func(t *testing.T) {
			t.Parallel()
			first, second := conformance.Subjects["accumulates"](), conformance.Subjects["accumulates"]()
			assert.NoError(t, first.Call(nil), "a call of accumulates succeeds")
			assert.Equal(t, first.Observe(), 1, "the first subject counts its call")
			assert.Equal(t, second.Observe(), 0, "the second subject counts none")
		})

		t.Run("returns a reads-handle that returns success for an absent handle", func(t *testing.T) {
			t.Parallel()
			var absent context.Context
			assert.NoError(
				t,
				conformance.Subjects["reads-handle"]().Ctx(absent, nil),
				"an absent handle gives no reason",
			)
		})

		t.Run("returns a rewrites-files that writes each file below the directory again", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{"docs/a.md": files.Text("# a\n")})
			path := filepath.Join(dir, "docs", "a.md")
			past := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
			assert.NoError(t, os.Chtimes(path, past, past), "the file is dated in the past")
			assert.NoError(t, conformance.Subjects["rewrites-files"]().Files(dir), "the subject writes the files")
			info, err := os.Stat(path)
			assert.NoError(t, err, "the file is there")
			assert.True(t, info.ModTime().After(past), "the subject writes the file again")
		})

		t.Run("returns a rewrites-files that fails on a file that its owner may not read", func(t *testing.T) {
			t.Parallel()
			mustRecordModes(t)
			dir := files.Workspace(t, files.Tree{"a.txt": files.Text("a").WithMode(0o200)})
			assert.ErrorIs(t, conformance.Subjects["rewrites-files"]().Files(dir), fs.ErrPermission,
				"the subject fails on the read")
		})

		t.Run("returns a settles-after that fails twice before it passes", func(t *testing.T) {
			t.Parallel()
			seated := conformance.Subjects["settles-after"]().Seated
			for attempt, want := range []bool{true, true, false} {
				r := assert.NewRecorder()
				seated(r)
				assert.Equal(t, r.Failed(), want, "attempt "+strconv.Itoa(attempt+1)+" of the subject")
			}
		})
	})
}
