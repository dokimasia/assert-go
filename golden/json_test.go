// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("MatchJSONField", func(t *testing.T) {
		t.Parallel()

		t.Run("a matching field reports nothing", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1,2],"two":["a"]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`[1,2]`), checking)

			assert.False(t, s.Failed(), "a field matching the golden file passes")
		})

		t.Run("formatting differences do not fail", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1,2]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte("[ 1,\n  2 ]"), checking)

			assert.False(t, s.Failed(),
				"both sides are re-encoded, so whitespace does not fail")
		})

		t.Run("a differing field reports a diff naming the field", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1,2]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`[3]`), checking)

			assert.True(t, s.Failed(), "a field differing from the golden file fails")
			assert.Contains(t, s.First(), `"one"`, "the failure names the field")
		})

		t.Run("a differing field reports a record of golden-match-json-field at the caller's line", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1,2]}`)
			s := &matchertest.Seat{}
			_, file, line, _ := runtime.Caller(0)
			golden.MatchJSONField(s, path, "one", []byte(`[3]`), checking)

			records := s.Records()
			assert.Length(t, records, 1, "the comparison reports one record")
			assert.Equal(t, records[0].Assertion, "golden-match-json-field", "the record names the comparison")
			assert.Equal(t, records[0].Detail,
				map[string]any{"want": "[\n  1,\n  2\n]", "got": "[\n  3\n]", "field": "one"},
				"the record states both values encoded, and the field")
			assert.Equal(t, records[0].Where, assert.Where{File: file, Line: line + 1},
				"the record names the line that called the comparison")
		})

		t.Run("a missing field names the flag that would add it", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "absent", []byte(`[]`), checking)

			assert.True(t, s.Failed(), "a field that is not there fails")
			assert.Contains(t, s.First(), "-update",
				"the failure names the flag that would add it")
			assert.Equal(t, s.Records()[0].Detail, map[string]any{"want": nil, "got": "[]", "field": "absent"},
				"the record states no golden value, the value, and the field")
		})

		t.Run("updating adds a field and keeps its siblings", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"kept":[9]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "added", []byte(`[1]`), updating)

			assert.False(t, s.Failed(), "updating a missing field passes")

			document := parse(t, path)
			assert.Contains(t, document, "kept", "the sibling field survives the update")
			assert.Contains(t, document, "added", "the named field was added")
		})

		t.Run("updating replaces one field and keeps its siblings", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1],"two":[2]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`[99]`), updating)

			assert.False(t, s.Failed(), "updating a differing field passes")

			document := parse(t, path)
			assert.Contains(t, document, "two", "the sibling field survives the update")
		})

		t.Run("a value that is not JSON reports", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`not json`), checking)

			assert.True(t, s.Failed(), "a value that is not JSON fails")
		})

		t.Run("a file that is not an object reports", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `[1,2,3]`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`[1]`), checking)

			assert.True(t, s.Failed(), "a golden file that is not an object fails")
		})

		t.Run("updating creates a missing file with the field alone", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "absent.json")
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`[1]`), updating)

			assert.False(t, s.Failed(), "updating a missing file passes")
			assert.Equal(t, parse(t, path), map[string]any{"one": []any{1.0}}, "the file contains the field alone")
		})

		t.Run("a missing file names the flag that would create it", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "absent.json")
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`[1]`), checking)

			assert.True(t, s.Failed(), "a golden file that does not exist fails")
			assert.Contains(t, s.First(), "-update",
				"the failure names the flag that would create it")
			assert.Equal(t, s.Records()[0].Detail, map[string]any{"want": nil, "got": "[\n  1\n]", "field": "one"},
				"the record states no golden value, the value, and the field")
		})
	})
}

// writtenJSON writes content to a golden file and returns its path.
func writtenJSON(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "golden.json")
	assert.NoError(t, os.WriteFile(path, []byte(content), goldenPerm),
		"the golden file for this case can be written")

	return path
}

// parse returns the object in a golden file.
func parse(t *testing.T, path string) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(path)
	assert.NoError(t, err, "the golden file can be read back")

	document := map[string]any{}
	assert.NoError(t, json.Unmarshal(raw, &document),
		"the golden file holds a JSON object")

	return document
}
