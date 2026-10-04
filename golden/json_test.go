// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("MatchJSONField", func(t *testing.T) {
		t.Parallel()

		t.Run("passes a field that matches the golden file", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1,2],"two":["a"]}`)
			r := assert.NewRecorder()
			golden.MatchJSONField(r, path, "one", []byte(`[1,2]`), checking)

			assert.False(t, r.Failed(), "a field matching the golden file passes")
			assert.Equal(t, verdicts(t, r), []string{"pass"}, "the call record of the pass")
		})

		t.Run("passes a field that differs from the golden file in formatting alone", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1,2]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte("[ 1,\n  2 ]"), checking)

			assert.False(t, s.Failed(),
				"both sides are re-encoded, so whitespace does not fail")
		})

		t.Run("fails a field that differs with a contract that states the field", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1,2]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`[3]`), checking)

			assert.True(t, s.Failed(), "a field differing from the golden file fails")
			assert.Contains(t, s.Records()[0].Contract, `"one"`, "the contract states the field")
		})

		t.Run(
			"reports a record of golden-match-json-field at the caller's line for a field that differs",
			func(t *testing.T) {
				t.Parallel()

				path := writtenJSON(t, `{"one":[1,2]}`)
				s := &matchertest.Seat{}
				_, file, line, _ := runtime.Caller(0)
				golden.MatchJSONField(s, path, "one", []byte(`[3]`), checking)

				records := s.Records()
				assert.Length(t, records, 1, "the comparison reports one record")
				assert.Equal(t, records[0].Assertion, "golden-match-json-field", "the record states the comparison")
				assert.Equal(t, records[0].Detail,
					map[string]any{"want": "[\n  1,\n  2\n]", "got": "[\n  3\n]", "field": "one"},
					"the record states both values encoded, and the field")
				assert.Equal(t, records[0].Where, assert.Where{File: file, Line: line + 1},
					"the record states the line that called the comparison")
			},
		)

		t.Run("fails a missing field with a record whose want is nil", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "absent", []byte(`[]`), checking)

			assert.True(t, s.Failed(), "a field that is not there fails")
			assert.Contains(t, s.Records()[0].Contract, "-update", "the contract states the flag that writes the field")
			assert.Equal(t, s.Records()[0].Detail, map[string]any{"want": nil, "got": "[]", "field": "absent"},
				"the record states no golden value, the value, and the field")
		})

		t.Run("adds a field, keeps the other fields and passes while updating", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"kept":[9]}`)
			r := assert.NewRecorder()
			golden.MatchJSONField(r, path, "added", []byte(`[1]`), updating)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating a missing field passes")
			assert.Equal(t, parse(t, path), map[string]any{"kept": []any{9.0}, "added": []any{1.0}},
				"the file contains the other field and the added one")
		})

		t.Run("replaces a field, keeps the other fields and passes while updating", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1],"two":[2]}`)
			r := assert.NewRecorder()
			golden.MatchJSONField(r, path, "one", []byte(`[99]`), updating)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating a differing field passes")
			assert.Equal(t, parse(t, path), map[string]any{"one": []any{99.0}, "two": []any{2.0}},
				"the file contains the other field and the replaced one")
		})

		t.Run("passes a matching field and leaves the file while updating", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"one":[1]}`)
			r := assert.NewRecorder()
			golden.MatchJSONField(r, path, "one", []byte(`[1]`), updating)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating a matching field passes")
			assert.Equal(t, read(t, path), `{"one":[1]}`, "the file is unchanged")
		})

		t.Run("writes a missing file with the field alone and passes while updating", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "absent.json")
			r := assert.NewRecorder()
			golden.MatchJSONField(r, path, "one", []byte(`[1]`), updating)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating a missing file passes")
			assert.Equal(t, parse(t, path), map[string]any{"one": []any{1.0}}, "the file contains the field alone")
		})

		t.Run("fails a missing file with a record whose want is nil", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "absent.json")
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "one", []byte(`[1]`), checking)

			assert.True(t, s.Failed(), "a golden file that does not exist fails")
			assert.Contains(t, s.Records()[0].Contract, "-update", "the contract states the flag that writes the file")
			assert.Equal(t, s.Records()[0].Detail, map[string]any{"want": nil, "got": "[\n  1\n]", "field": "one"},
				"the record states no golden value, the value, and the field")
		})

		t.Run("fails a field whose number differs past the precision of a float64", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"n":9007199254740992}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "n", []byte(`9007199254740993`), checking)

			assert.True(t, s.Failed(), "2^53 + 1 does not match 2^53")
			assert.Equal(t, s.Records()[0].Detail,
				map[string]any{"want": "9007199254740992", "got": "9007199254740993", "field": "n"},
				"the record states both numbers exactly")
		})

		t.Run("passes a field whose numbers differ in their text alone", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"n":[1.0,1e2,-0,0.50,1.5E-3]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "n", []byte(`[1,100,0,0.5,0.0015]`), checking)

			assert.False(t, s.Failed(), "each number compares by the value its text states")
		})

		t.Run("passes an object field whose negative numbers differ in their text alone", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"o":{"a":-1.50,"b":[-2e0]}}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "o", []byte(`{"b":[-2],"a":-1.5}`), checking)

			assert.False(t, s.Failed(), "each number of the object compares by the value its text states")
		})

		t.Run("states a number from 10^1000 or below 10^-1001 with an exponent", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"n":[10e1000,15e-1003,1e999,1e-1001]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "n", []byte(`[2e1001,1.5e-1002,1e999,1e-1001]`), checking)

			assert.True(t, s.Failed(), "2e1001 does not match 1e1001")
			between := "1" + strings.Repeat("0", 999) + ",\n  0." + strings.Repeat("0", 1000) + "1"
			assert.Equal(t, s.Records()[0].Detail, map[string]any{
				"want":  "[\n  1e1001,\n  1.5e-1002,\n  " + between + "\n]",
				"got":   "[\n  2e1001,\n  1.5e-1002,\n  " + between + "\n]",
				"field": "n",
			}, "the record states an exponent from 10^1000 and below 10^-1001, and every digit between")
		})

		t.Run("compares a number whose exponent does not fit 32 bits by its text", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"n":[1e2147483649,1.50]}`)
			s := &matchertest.Seat{}
			golden.MatchJSONField(s, path, "n", []byte(`[10e2147483648,1.5]`), checking)

			assert.True(t, s.Failed(), "two texts of one value differ beyond 32 bits of exponent")
			assert.Equal(t, s.Records()[0].Detail, map[string]any{
				"want":  "[\n  1e2147483649,\n  1.5\n]",
				"got":   "[\n  10e2147483648,\n  1.5\n]",
				"field": "n",
			}, "the record states the text of each number beyond 32 bits, and the exact text of every other")
		})

		t.Run("keeps the text of every other field while updating", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{"big":9007199254740993,"one":[1]}`)
			r := assert.NewRecorder()
			golden.MatchJSONField(r, path, "one", []byte(`[2]`), updating)

			assert.Equal(t, verdicts(t, r), []string{"pass"}, "updating a differing field passes")
			assert.Equal(t, read(t, path), "{\n  \"big\": 9007199254740993,\n  \"one\": [\n    2\n  ]\n}\n",
				"the file states the other field's number as it was")
		})

		t.Run("keeps every field that parallel calls add while updating", func(t *testing.T) {
			t.Parallel()

			path := writtenJSON(t, `{}`)
			const writers = 32
			var wg sync.WaitGroup
			for i := range writers {
				wg.Go(func() {
					golden.MatchJSONField(assert.NewRecorder(), path, fmt.Sprintf("f%02d", i), []byte(strconv.Itoa(i)),
						updating)
				})
			}
			wg.Wait()
			assert.Length(t, parse(t, path), writers, "the file contains the field of every call")
		})

		tests := []struct {
			name       string
			givePath   func(t *testing.T) string
			giveValue  string
			giveUpdate bool
		}{
			{
				name:      "ends with a fault for a value that is no JSON",
				givePath:  func(t *testing.T) string { t.Helper(); return writtenJSON(t, `{"one":[1]}`) },
				giveValue: `not json`,
			},
			{
				name:      "ends with a fault for a golden file that is no JSON object",
				givePath:  func(t *testing.T) string { t.Helper(); return writtenJSON(t, `[1,2,3]`) },
				giveValue: `[1]`,
			},
			{
				name:       "ends with a fault for a golden file that is null while updating",
				givePath:   func(t *testing.T) string { t.Helper(); return writtenJSON(t, `null`) },
				giveValue:  `[1]`,
				giveUpdate: updating,
			},
			{
				name:      "ends with a fault for a value followed by more data",
				givePath:  func(t *testing.T) string { t.Helper(); return writtenJSON(t, `{"one":[1]}`) },
				giveValue: `[1] [2]`,
			},
			{
				name:      "ends with a fault for a golden file that cannot be read",
				givePath:  func(t *testing.T) string { t.Helper(); return t.TempDir() },
				giveValue: `[1]`,
			},
			{
				name:       "ends with a fault for a golden file that cannot be written while updating",
				givePath:   dangling,
				giveValue:  `[1]`,
				giveUpdate: updating,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				r := assert.NewRecorder()
				golden.MatchJSONField(r, tt.givePath(t), "one", []byte(tt.giveValue), tt.giveUpdate)

				assert.True(t, r.Failed(), "the fault stops the test")
				assert.Length(t, r.Failures(), 0, "the call reports no failure record")
				assert.Equal(t, verdicts(t, r), []string{"error"}, "the call record states the fault")
			})
		}
	})
}

// TestJSONAllocs checks the allocation ceiling of a passing call of
// MatchJSONField.
func TestJSONAllocs(t *testing.T) {
	alloctest.Check(t, jsonCases(t, t.TempDir()))
}

// BenchmarkJSON measures a passing call of MatchJSONField.
func BenchmarkJSON(b *testing.B) {
	for _, c := range jsonCases(b, b.TempDir()) {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// jsonCases returns a passing call of MatchJSONField, with its allocation
// ceiling, measured. It writes the golden file of the call into dir.
func jsonCases(tb testing.TB, dir string) []alloctest.Case {
	tb.Helper()
	path := filepath.Join(dir, "fields.json")
	assert.NoError(tb, os.WriteFile(path, []byte(`{"count": 3}`), goldenPerm), "the golden file is written")
	got := []byte(`3`)
	return []alloctest.Case{{
		Name:   "MatchJSONField",
		Call:   func(tb assert.TB) { golden.MatchJSONField(tb, path, "count", got, checking) },
		Allocs: 41,
	}}
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
		"the golden file contains a JSON object")

	return document
}
