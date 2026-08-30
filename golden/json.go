// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden

import (
	"encoding/json"
	"fmt"
	"os"

	"go.dokimi.dev/assert"
)

// jsonIndent is how this package writes a golden JSON file, so a diff
// between two of them reads line by line.
const jsonIndent = "  "

// MatchJSONField compares got against one named field of the JSON
// object at path, taken as given.
//
// Use it where one golden file holds several independent values, one
// per field. Each test then compares only its own, so a failure shows
// that value's diff rather than the whole file's, and two tests
// updating different fields do not overwrite each other.
//
// Comparison is structural: both sides are re-encoded with the same
// indentation first, so formatting differences do not fail. got must
// be valid JSON.
//
// A missing file or a missing field behaves as [Match] does for a
// missing file: a failure naming -update, or the field written and the
// siblings left alone.
func MatchJSONField(tb assert.TB, path, field string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()

	var value any
	assert.NoError(tb, json.Unmarshal(got, &value),
		fmt.Sprintf("%s: the value given for field %q is valid JSON", path, field))

	document, ok := readObject(tb, path, update)
	if !ok {
		return
	}

	held, present := document[field]
	if !present {
		if !update {
			tb.Fatalf("%s: the golden file has no field %q; "+
				"run the test with -update to add it", path, field)
			return
		}
		document[field] = value
		writeObject(tb, path, document)
		return
	}

	mine := scrub(encode(tb, value, path, field), scrubbers)
	theirs := scrub(encode(tb, held, path, field), scrubbers)

	if update {
		if mine != theirs {
			document[field] = value
			writeObject(tb, path, document)
		}
		return
	}

	assert.Equal(tb, mine, theirs,
		fmt.Sprintf("%s: field %q matches the golden file; "+
			"read the diff before running with -update", path, field))
}

// readObject reads the JSON object at path, reporting whether the
// caller may carry on. It answers a fresh object when the file is
// missing and update is set.
func readObject(tb assert.TB, path string, update bool) (map[string]any, bool) {
	tb.Helper()

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if !update {
			tb.Fatalf("%s: the golden file does not exist; "+
				"run the test with -update to create it", path)
			return nil, false
		}
		return map[string]any{}, true
	}
	assert.NoError(tb, err, fmt.Sprintf("%s: the golden file can be read", path))

	document := map[string]any{}
	assert.NoError(tb, json.Unmarshal(raw, &document),
		fmt.Sprintf("%s: the golden file is a JSON object", path))

	return document, true
}

// writeObject records document as the golden file at path.
func writeObject(tb assert.TB, path string, document map[string]any) {
	tb.Helper()

	raw, err := json.MarshalIndent(document, "", jsonIndent)
	assert.NoError(tb, err, fmt.Sprintf("%s: the golden file can be encoded", path))

	write(tb, path, string(raw)+"\n")
}

// encode renders one field's value with the indentation both sides of
// a comparison use.
func encode(tb assert.TB, value any, path, field string) string {
	tb.Helper()

	raw, err := json.MarshalIndent(value, "", jsonIndent)
	assert.NoError(tb, err,
		fmt.Sprintf("%s: the value for field %q can be encoded", path, field))

	return string(raw)
}
