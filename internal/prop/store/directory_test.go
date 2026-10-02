// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/store"
)

// The allocations of the directory functions, measured.
const (
	// loadAllocs are the allocations of Load on a directory of one entry.
	loadAllocs = 149
	// saveAllocs are the allocations of Save of an entry whose file exists.
	saveAllocs = 152
)

// The modes that the tests set and check.
const (
	// fileMode is the mode of an entry's file before the umask.
	fileMode = 0o644
	// ownerMode is the part of a file's mode that no umask takes away.
	ownerMode = 0o600
	// readOnly is the mode of a directory that no file can be created in.
	readOnly = 0o555
	// writable is the mode that restores a directory for its removal.
	writable = 0o755
)

// TestDirectory checks how a run reads its store and writes an entry.
func TestDirectory(t *testing.T) {
	t.Parallel()

	t.Run("Load", func(t *testing.T) {
		t.Parallel()

		t.Run("returns an empty store for a directory that does not exist", func(t *testing.T) {
			t.Parallel()
			got, err := store.Load(filepath.Join(t.TempDir(), "absent"), contract)
			assert.NoError(t, err, "no store is an empty store")
			assert.Equal(t, got, store.Stored{}, "nothing found")
		})

		t.Run("returns the entries oldest first and by name within a date", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, "b.json", entryText(t, map[string]any{"choices": "prop1:AAc", "found": "2026-10-01"}))
			write(t, dir, "a.json", entryText(t, map[string]any{"choices": "prop1:AAM", "found": "2026-10-01"}))
			write(t, dir, "c.json", entryText(t, map[string]any{"choices": "prop1:AAE", "found": "2026-09-30"}))
			got, err := store.Load(dir, contract)
			assert.NoError(t, err, "every file is an entry")
			assert.Equal(t, firstChoices(got.Entries), []int64{1, 3, 7}, "c, then a, then b")
		})

		t.Run("returns a note for each skipped file", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, "later.json", `{"store": 2}`)
			got, err := store.Load(dir, contract)
			assert.NoError(t, err, "a later format is no damage")
			want := []string{"later.json: store: later than this reader: the entry is of format 2"}
			assert.Equal(t, got.Skipped, want, "the file and the reason")
		})

		t.Run("leaves out the entries of another property", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, "other.json", entryText(t, map[string]any{"property": "encoding is deterministic"}))
			got, err := store.Load(dir, contract)
			assert.NoError(t, err, "an entry of another property is no damage")
			assert.Equal(t, got, store.Stored{}, "nothing for this property")
		})

		t.Run("leaves out files whose names do not end in .json and directories", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, "notes.txt", "not an entry")
			assert.NoError(t, os.Mkdir(filepath.Join(dir, "nested.json"), writable), "a directory named like an entry")
			got, err := store.Load(dir, contract)
			assert.NoError(t, err, "neither is read")
			assert.Equal(t, got, store.Stored{}, "nothing found")
		})

		t.Run("returns ErrDamaged naming each damaged file after reading the others", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, "broken.json", "{")
			write(t, dir, "empty.json", "")
			write(t, dir, "good.json", entryText(t, nil))
			got, err := store.Load(dir, contract)
			assert.ErrorIs(t, err, store.ErrDamaged, "the damage")
			assert.Contains(t, err.Error(), "store: read broken.json: store: not an entry", "the first file")
			assert.Contains(t, err.Error(), "store: read empty.json: store: not an entry", "the second file")
			assert.Equal(t, firstChoices(got.Entries), []int64{7}, "the entry that is not damaged")
		})

		t.Run("returns the error of a file that cannot be read", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			assert.NoError(t, os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "gone.json")),
				"a link to nothing")
			_, err := store.Load(dir, contract)
			assert.ErrorIs(t, err, fs.ErrNotExist, "the link's target is missing")
			assert.False(t, errors.Is(err, store.ErrDamaged), "no damage")
		})

		t.Run("returns the error of a directory that cannot be read", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "file")
			write(t, filepath.Dir(path), "file", "")
			_, err := store.Load(path, contract)
			assert.HasError(t, err, "a file is no directory")
			assert.True(t, strings.HasPrefix(err.Error(), "store: "), "the package's prefix")
		})
	})

	t.Run("Save", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the entry under its name in a new directory", func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "testdata", "prop", "TestCodec")
			wrote, err := store.Save(dir, pinned())
			assert.NoError(t, err, "the entry is written")
			assert.True(t, wrote, "a new file")
			data, err := os.ReadFile(filepath.Join(dir, pinnedName))
			assert.NoError(t, err, "the file has the entry's name")
			var want, got any
			assert.NoError(t, json.Unmarshal([]byte(pinnedJSON), &want), "the pinned entry is JSON")
			assert.NoError(t, json.Unmarshal(data, &got), "the file is JSON")
			assert.Equal(t, got, want, "the pinned entry")
			assert.True(t, strings.HasPrefix(string(data), "{\n  \"store\": 1,\n"), "indented by two spaces")
			assert.True(t, strings.HasSuffix(string(data), "}\n"), "a final newline")
		})

		t.Run("writes a file that the next run replays", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			_, err := store.Save(dir, pinned())
			assert.NoError(t, err, "the entry is written")
			got, err := store.Load(dir, contract)
			assert.NoError(t, err, "the store is read")
			assert.Equal(t, len(got.Entries), 1, "one entry")
			assert.True(t, sameChoices(got.Entries[0].Choices, pinned().Choices), "its choices")
		})

		t.Run("writes the file with mode 0o644 less the umask", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			_, err := store.Save(dir, pinned())
			assert.NoError(t, err, "the entry is written")
			info, err := os.Stat(filepath.Join(dir, pinnedName))
			assert.NoError(t, err, "the file exists")
			assert.Equal(t, info.Mode().Perm()&^fileMode, fs.FileMode(0), "no permission beyond 0o644")
			assert.Equal(t, info.Mode().Perm()&ownerMode, fs.FileMode(ownerMode), "the owner reads and writes")
		})

		t.Run("returns false and keeps the file when one of the name exists", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			_, err := store.Save(dir, pinned())
			assert.NoError(t, err, "the first entry is written")
			later := pinned()
			later.Found = later.Found.Add(48 * time.Hour)
			wrote, err := store.Save(dir, later)
			assert.NoError(t, err, "an existing entry is no error")
			assert.False(t, wrote, "no file written")
			data, err := os.ReadFile(filepath.Join(dir, pinnedName))
			assert.NoError(t, err, "the file exists")
			assert.Contains(t, string(data), `"found": "2026-10-01"`, "the first entry's date")
		})

		t.Run("returns ErrInvalid for an entry that a reader would not replay", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			invalid := pinned()
			invalid.Identity = store.Identity{Assertion: "equal"}
			wrote, err := store.Save(dir, invalid)
			assert.ErrorIs(t, err, store.ErrInvalid, "the refusal")
			assert.Contains(t, err.Error(), "has no shape", "the fault")
			assert.False(t, wrote, "no file written")
			entries, err := os.ReadDir(dir)
			assert.NoError(t, err, "the directory is read")
			assert.Empty(t, entries, "nothing written")
		})

		t.Run("returns ErrInvalid for a value that is not JSON", func(t *testing.T) {
			t.Parallel()
			broken := pinned()
			broken.Counterexample = []store.Draw{{Label: "broken", Value: json.RawMessage("{")}}
			wrote, err := store.Save(t.TempDir(), broken)
			assert.ErrorIs(t, err, store.ErrInvalid, "the refusal")
			assert.False(t, wrote, "no file written")
		})

		t.Run("returns the error of a directory that cannot be created", func(t *testing.T) {
			t.Parallel()
			parent := t.TempDir()
			write(t, parent, "file", "")
			wrote, err := store.Save(filepath.Join(parent, "file", "prop"), pinned())
			assert.HasError(t, err, "a file is no directory")
			assert.True(t, strings.HasPrefix(err.Error(), "store: "), "the package's prefix")
			assert.False(t, wrote, "no file written")
		})

		t.Run("returns the error of a directory that cannot be written", func(t *testing.T) {
			t.Parallel()
			if os.Geteuid() == 0 {
				t.Skip("root writes into a read-only directory")
			}
			dir := t.TempDir()
			assert.NoError(t, os.Chmod(dir, readOnly), "the directory is read-only")
			t.Cleanup(func() { _ = os.Chmod(dir, writable) })
			wrote, err := store.Save(dir, pinned())
			assert.ErrorIs(t, err, fs.ErrPermission, "no file can be created")
			assert.True(t, strings.HasPrefix(err.Error(), "store: "), "the package's prefix")
			assert.False(t, wrote, "no file written")
		})
	})
}

// TestDirectoryZeroAlloc checks the ceilings of Load on a directory of one
// entry and of Save of an entry whose file exists.
func TestDirectoryZeroAlloc(t *testing.T) {
	dir := t.TempDir()
	e := pinned()
	_, err := store.Save(dir, e)
	assert.NoError(t, err, "the entry is written")
	assert.MaxAllocs(t, func() { _, _ = store.Load(dir, contract) }, loadAllocs, "Load allocates what it reads")
	assert.MaxAllocs(t, func() { _, _ = store.Save(dir, e) }, saveAllocs, "Save allocates what it writes")
}

// BenchmarkDirectory measures Load on a directory of one entry, and Save of
// an entry whose file exists.
func BenchmarkDirectory(b *testing.B) {
	dir := b.TempDir()
	_, err := store.Save(dir, pinned())
	assert.NoError(b, err, "the entry is written")

	b.Run("Load", func(b *testing.B) {
		var got store.Stored
		c := bench.Start(b).MaxAllocs(loadAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = store.Load(dir, contract)
		}
		assert.Equal(b, len(got.Entries), 1, "the one entry")
	})

	b.Run("Save", func(b *testing.B) {
		var got bool
		e := pinned()
		c := bench.Start(b).MaxAllocs(saveAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = store.Save(dir, e)
		}
		assert.False(b, got, "the file exists")
	})
}

// write writes text to the file name of dir, failing the test when it
// cannot.
func write(tb testing.TB, dir, name, text string) {
	tb.Helper()
	assert.NoError(tb, os.WriteFile(filepath.Join(dir, name), []byte(text), fileMode), "the file is written")
}

// firstChoices returns the value of the first choice of each entry, in
// order.
func firstChoices(entries []store.Entry) []int64 {
	out := make([]int64, len(entries))
	for i, e := range entries {
		out[i], _ = e.Choices[0].Integer.Int64()
	}
	return out
}
