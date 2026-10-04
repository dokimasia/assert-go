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
// Use it where one golden file contains independent values, one per
// field. Each test then compares only its own field, so a failure
// shows the diff of that value alone, and two tests that update
// different fields do not overwrite each other.
//
// Comparison is structural: both sides are re-encoded with the same
// indentation first, so formatting differences do not fail.
//
// A missing file or a missing field behaves as [Match] does for a
// missing file: the call fails while update is false, and update writes
// the field, leaves the other fields as they are, and passes. A failure is
// a record of golden-match-json-field, with the field's golden value as
// want, nil when it is missing, the value as got, both encoded and
// scrubbed, and the field's name.
//
// A got that is no JSON, a golden file that is no JSON object, and a
// golden file that cannot be read or written end the call with a fault,
// which stops the test.
//
// # Allocation contract
//
// A passing comparison of an integer field in a document of one field
// allocates 24 times. A larger document allocates more, because the
// comparison decodes the whole document.
func MatchJSONField(tb assert.TB, path, field string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	c := call{
		tb: tb, op: "golden.MatchJSONField", id: jsonFieldID, path: path,
		contract: fmt.Sprintf("the field %q of the golden file %s matches the value, and -update writes it",
			field, path),
	}

	var value any
	if err := json.Unmarshal(got, &value); err != nil {
		c.fault(err, "the value of the field %q is no JSON", field)
		return
	}
	mine := scrub(encode(value), scrubbers)

	document, ok := c.readObject(field, mine, update)
	if !ok {
		return
	}

	stored, present := document[field]
	if !present {
		if !update {
			c.fail(map[string]any{"want": nil, "got": mine, "field": field})
			return
		}
		document[field] = value
		c.writeObject(document)
		return
	}

	theirs := scrub(encode(stored), scrubbers)
	if mine == theirs {
		c.pass()
		return
	}
	if update {
		document[field] = value
		c.writeObject(document)
		return
	}
	c.fail(map[string]any{"want": theirs, "got": mine, "field": field})
}

// readObject reads the JSON object of the golden file of c, and reports
// whether the call continues. It returns an empty object for a missing
// file while update is true. For a missing file without update, it reports
// the failure of field, with got as mine.
func (c call) readObject(field, mine string, update bool) (map[string]any, bool) {
	c.tb.Helper()

	raw, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		if !update {
			c.fail(map[string]any{"want": nil, "got": mine, "field": field})
			return nil, false
		}
		return map[string]any{}, true
	}
	document := map[string]any{}
	if err == nil {
		err = json.Unmarshal(raw, &document)
	}
	if err != nil {
		c.fault(err, "the golden file cannot be read as a JSON object")
		return nil, false
	}
	return document, true
}

// writeObject writes document as the golden file of c, and passes.
func (c call) writeObject(document map[string]any) {
	c.tb.Helper()
	c.write(encode(document) + "\n")
}

// encode returns the indented JSON of value, which json.Unmarshal decoded
// or which contains only values that it decoded. MarshalIndent returns no
// error for such a value, whose numbers are finite and whose strings are
// UTF-8.
func encode(value any) string {
	raw, _ := json.MarshalIndent(value, "", jsonIndent)
	return string(raw)
}
