// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"go.dokimi.dev/assert"
)

// jsonIndent is how this package writes a golden JSON file, so a diff
// between two of them reads line by line.
const jsonIndent = "  "

// maxNumberDigits bounds the numbers that the comparison states without an
// exponent: those below 10^1000 and at least 10^-1001 in magnitude. Every
// other number keeps an exponent, so the text of 1e999999999 has 11
// characters.
const maxNumberDigits = 1000

// errTrailing is the cause of a value that is followed by more data.
var errTrailing = errors.New("golden: the JSON value is followed by more data")

// files are the locks of the golden JSON files that this test process
// compares, by the path of each file that [identity] returns.
// [MatchJSONField] locks a file from its read to its write.
var files sync.Map

// paths are the locks of files by each absolute path that a call named, so
// that a later call on a path follows no link. A path keeps the lock of the
// file that it led to at its first call.
var paths sync.Map

// MatchJSONField compares got against one named field of the JSON
// object at path, taken as given.
//
// Use it where one golden file contains independent values, one per
// field. Each test then compares only its own field, so a failure
// shows the diff of that value alone, and two tests that update
// different fields keep both, also when they run in parallel.
//
// Comparison is structural: both sides are re-encoded with the same
// indentation first, so formatting differences do not fail. A number
// compares by the exact value that its text states, so 1.0 matches 1 and
// 9007199254740993 does not match 9007199254740992. A number whose exponent
// does not fit 32 bits compares by its text.
//
// A missing file or a missing field behaves as [Match] does for a
// missing file: the call fails while update is false, and update writes
// the field, leaves the text of every other field as it is, and passes. A
// failure is a record of golden-match-json-field, with the field's golden
// value as want, nil when it is missing, the value as got, both encoded
// and scrubbed, and the field's name.
//
// A got that is no JSON, a golden file that is no JSON object, and a
// golden file that cannot be read or written end the call with a fault,
// which stops the test.
//
// # Concurrency
//
// Calls on one golden file from tests of one process run one at a time,
// from the read of the file to its write. Two paths name one golden file
// when they lead to it through links, such as a link to the file or to a
// directory above it. A path keeps the file that it led to at the first
// call on it.
//
// # Allocation contract
//
// A passing comparison of an integer field in a document of one field
// allocates 41 times. A larger document allocates more, because the
// comparison reads the whole document.
func MatchJSONField(tb assert.TB, path, field string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	c := call{
		tb: tb, op: "golden.MatchJSONField", id: jsonFieldID, path: path,
		contract: fmt.Sprintf("the field %q of the golden file %s matches the value, and -update writes it",
			field, path),
	}

	value, err := decodeNumbers(got)
	if err != nil {
		c.fault(err, "the value of the field %q is no JSON", field)
		return
	}
	mine := scrub(encodeExact(value), scrubbers)

	defer lock(path)()
	document, ok := c.readObject()
	if !ok {
		return
	}

	stored, present := document[field]
	if !present {
		if !update {
			c.fail(map[string]any{"want": nil, "got": mine, "field": field})
			return
		}
		document[field] = got
		c.writeObject(document)
		return
	}

	// The document decoded, so each of its fields is a JSON value.
	theirsValue, _ := decodeNumbers(stored)
	theirs := scrub(encodeExact(theirsValue), scrubbers)
	if mine == theirs {
		c.pass()
		return
	}
	if update {
		document[field] = got
		c.writeObject(document)
		return
	}
	c.fail(map[string]any{"want": theirs, "got": mine, "field": field})
}

// lock locks the golden file at path for this process, and returns the
// function that unlocks it.
func lock(path string) func() {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	stored, ok := paths.Load(path)
	if !ok {
		shared, _ := files.LoadOrStore(identity(path), new(sync.Mutex))
		stored, _ = paths.LoadOrStore(path, shared)
	}
	mu := stored.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// identity returns the path that every path of the golden file at path, an
// absolute path, resolves to: path with each link followed, and for a file
// that does not exist yet, its directory's path with each link followed,
// joined with its name. A file whose directory does not exist yet keeps
// path.
func identity(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	if dir, err := filepath.EvalSymlinks(filepath.Dir(path)); err == nil {
		return filepath.Join(dir, filepath.Base(path))
	}
	return path
}

// readObject reads the JSON object of the golden file of c, with the text of
// each field, and reports whether the call continues. It returns an empty
// object for a missing file.
func (c call) readObject() (map[string]json.RawMessage, bool) {
	c.tb.Helper()

	raw, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return map[string]json.RawMessage{}, true
	}
	var document map[string]json.RawMessage
	if err == nil {
		err = json.Unmarshal(raw, &document)
	}
	// Unmarshal leaves document nil for every file but one of a JSON object.
	if document == nil {
		c.fault(err, "the golden file cannot be read as a JSON object")
		return nil, false
	}
	return document, true
}

// writeObject writes document as the golden file of c, and passes.
func (c call) writeObject(document map[string]json.RawMessage) {
	c.tb.Helper()
	c.write(encode(document) + "\n")
}

// decodeNumbers returns the value of the one JSON value in raw, with each
// number as the json.Number of its text.
func decodeNumbers(raw []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value any
	if err := d.Decode(&value); err != nil {
		return nil, err
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return nil, errTrailing
	}
	return value, nil
}

// encodeExact returns the indented JSON of a value that decodeNumbers
// returned, with each number in the one text of its exact value.
func encodeExact(value any) string {
	return encode(exact(value))
}

// exact returns value with each number in the one text of its exact value,
// and changes the maps and the slices of value in place.
func exact(value any) any {
	switch v := value.(type) {
	case json.Number:
		return exactNumber(v)
	case []any:
		for i := range v {
			v[i] = exact(v[i])
		}
	case map[string]any:
		for key := range v {
			v[key] = exact(v[key])
		}
	}
	return value
}

// exactNumber returns the one text of the value that n states: its digits
// without leading zeros in the integer part or trailing zeros in the
// fraction, without an exponent, and 0 for a zero of either sign. A number
// beyond the bounds of maxNumberDigits keeps one digit before the point and
// an exponent. n is the text of a JSON number, and a number whose exponent
// does not fit 32 bits keeps the text of n.
func exactNumber(n json.Number) json.Number {
	text := string(n)
	negative := strings.HasPrefix(text, "-")
	text = strings.TrimPrefix(text, "-")
	mantissa, exponent, scientific := strings.Cut(strings.ToLower(text), "e")
	var shift int64
	if scientific {
		parsed, err := strconv.ParseInt(exponent, 10, 32)
		if err != nil {
			return n
		}
		shift = parsed
	}
	whole, fraction, _ := strings.Cut(mantissa, ".")
	digits := whole + fraction
	trimmed := strings.TrimLeft(digits, "0")
	// The value is 0.<digits> × 10^point once the zeros are trimmed.
	point := int64(len(whole)-(len(digits)-len(trimmed))) + shift
	digits = strings.TrimRight(trimmed, "0")
	if digits == "" {
		return "0"
	}
	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	switch {
	case point > maxNumberDigits || point < -maxNumberDigits:
		b.WriteString(digits[:1])
		if len(digits) > 1 {
			b.WriteString("." + digits[1:])
		}
		b.WriteString("e" + strconv.FormatInt(point-1, 10))
	case point <= 0:
		b.WriteString("0." + strings.Repeat("0", int(-point)) + digits)
	case point >= int64(len(digits)):
		b.WriteString(digits + strings.Repeat("0", int(point)-len(digits)))
	default:
		b.WriteString(digits[:point] + "." + digits[point:])
	}
	return json.Number(b.String())
}

// encode returns the indented JSON of value, which json.Unmarshal decoded
// or which contains only values that it decoded. MarshalIndent returns no
// error for such a value, whose numbers are finite and whose strings are
// UTF-8.
func encode(value any) string {
	raw, _ := json.MarshalIndent(value, "", jsonIndent)
	return string(raw)
}
