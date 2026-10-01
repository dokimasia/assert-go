// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/prop/token"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Verdict -linecomment -output=read.string_gen.go

// Verdict is what a run does with one file of its store.
type Verdict uint8

const (
	// Replay is an entry of [Format] for the property, whose choices the
	// run tries first.
	Replay Verdict = 0 // replay
	// Other is an entry of [Format] for another property of the test,
	// which the run leaves as it is.
	Other Verdict = 1 // other
	// Skip is an entry of a later format, or whose token is of a later
	// version, which the run notes and passes over.
	Skip Verdict = 2 // skip
	// Damaged is a file that is no entry. It fails the test.
	Damaged Verdict = 3 // damaged
)

// Valid reports whether v is one of the four verdicts.
func (v Verdict) Valid() bool {
	return v <= Damaged
}

var (
	// ErrDamaged reports a file of a store that is no entry, whose verdict
	// is [Damaged].
	ErrDamaged = errors.New("store: not an entry")
	// ErrLater reports an entry of a later format, or with a token of a
	// later version, whose verdict is [Skip].
	ErrLater = errors.New("store: later than this reader")
)

// The names of an entry's fields.
const (
	// fieldStore is the format of the entry.
	fieldStore = "store"
	// fieldDefinition is the definition version that wrote the entry.
	fieldDefinition = "definition"
	// fieldProperty is the property's contract.
	fieldProperty = "property"
	// fieldIdentity is the identity of the failure.
	fieldIdentity = "identity"
	// fieldChoices is the replay token of the case.
	fieldChoices = "choices"
	// fieldCounterexample is the list of drawn values.
	fieldCounterexample = "counterexample"
	// fieldFound is the date on which the run wrote the entry.
	fieldFound = "found"
)

// The names of the keys of an identity and of a draw.
const (
	// keyAssertion is an identity's assertion.
	keyAssertion = "assertion"
	// keyContract is an identity's contract.
	keyContract = "contract"
	// keyError is an identity's error type.
	keyError = "error"
	// keyFile is an identity's base name.
	keyFile = "file"
	// keyLine is an identity's line.
	keyLine = "line"
	// keyLabel is a draw's label.
	keyLabel = "label"
	// keyValue is a draw's value.
	keyValue = "value"
)

// The delimiters of JSON objects and arrays.
const (
	// openObject starts an object.
	openObject = json.Delim('{')
	// closeObject ends an object.
	closeObject = json.Delim('}')
	// openArray starts an array.
	openArray = json.Delim('[')
)

// currentToken is the version of the tokens that package token reads.
const currentToken = "1"

// fields are the fields of an entry of Format, in the order that the
// definition lists them.
var fields = []string{
	fieldStore, fieldDefinition, fieldProperty, fieldIdentity, fieldChoices, fieldCounterexample, fieldFound,
}

// The forms of an entry's texts.
var (
	// version is the form of a definition version, MAJOR.MINOR.PATCH.
	version = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	// positive is the form of a positive JSON integer.
	positive = regexp.MustCompile(`^[1-9][0-9]*$`)
	// tokenVersion is the start of a token, which states its version.
	tokenVersion = regexp.MustCompile(`^prop([1-9][0-9]*):`)
)

// container is an object or an array that the walk of a file is inside.
type container struct {
	// names are an object's names so far, and nil for an array.
	names map[string]struct{}
	// atName reports whether an object's next token is a name or its end.
	atName bool
}

// Read returns the verdict on the content of one file for the property
// contract, with the entry that the file states for [Replay] and [Other].
// It returns an error that wraps [ErrLater] for [Skip], and one that wraps
// [ErrDamaged] for [Damaged], each naming the fault.
//
// The checks run in the definition's order: the file's JSON, the format,
// the fields and their forms, and then the token's version and its
// choices. A file of a later format is skipped whatever its other fields
// state.
func Read(data []byte, contract string) (Entry, Verdict, error) {
	e, err := decode(data)
	if errors.Is(err, ErrLater) {
		return Entry{}, Skip, err
	}
	if err != nil {
		return Entry{}, Damaged, err
	}
	if e.Property != contract {
		return e, Other, nil
	}
	return e, Replay, nil
}

// decode returns the entry that data states. It returns an error that
// wraps ErrLater for an entry of a later format or token version, and one
// that wraps ErrDamaged when data is no entry of Format.
func decode(data []byte) (Entry, error) {
	if err := wellFormed(data); err != nil {
		return Entry{}, fmt.Errorf("%w: %w", ErrDamaged, err)
	}
	root, ok := members(data)
	if !ok {
		return Entry{}, fmt.Errorf("%w: the file is not one JSON object", ErrDamaged)
	}
	format := root[fieldStore]
	if !positive.Match(format) {
		return Entry{}, fmt.Errorf("%w: the entry states no format as store", ErrDamaged)
	}
	if string(format) != strconv.Itoa(Format) {
		return Entry{}, fmt.Errorf("%w: the entry is of format %s", ErrLater, format)
	}
	e, tok, err := entryOf(root)
	if err != nil {
		return Entry{}, fmt.Errorf("%w: %w", ErrDamaged, err)
	}
	stated := tokenVersion.FindStringSubmatch(tok)
	if stated == nil {
		return Entry{}, fmt.Errorf("%w: choices %q states no token version", ErrDamaged, tok)
	}
	if stated[1] != currentToken {
		return Entry{}, fmt.Errorf("%w: the token is of version %s", ErrLater, stated[1])
	}
	if e.Choices, err = token.Decode(tok); err != nil {
		return Entry{}, fmt.Errorf("%w: %w", ErrDamaged, err)
	}
	return e, nil
}

// wellFormed returns why data is no JSON value that every reader reads
// alike, and nil when it is one: UTF-8 text of one value, no name that
// repeats within an object, and at most maxDepth levels of objects and
// arrays.
func wellFormed(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("the file is not UTF-8 from byte %d", invalidAt(data))
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var open []*container
	values := 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if n := len(open); n > 0 && open[n-1].atName {
			if tok == closeObject {
				open = ended(open[:n-1])
				continue
			}
			name, _ := tok.(string)
			if _, seen := open[n-1].names[name]; seen {
				return fmt.Errorf("an object repeats the name %q", name)
			}
			open[n-1].names[name], open[n-1].atName = struct{}{}, false
			continue
		}
		if len(open) == 0 {
			values++
		}
		if tok == openObject {
			open = append(open, &container{names: make(map[string]struct{}), atName: true})
		} else if tok == openArray {
			open = append(open, &container{})
		} else if _, closes := tok.(json.Delim); closes {
			open = ended(open[:len(open)-1])
		} else {
			open = ended(open)
		}
		if len(open) > maxDepth {
			return fmt.Errorf("the file nests past %d levels", maxDepth)
		}
	}
	if values != 1 {
		return fmt.Errorf("the file states %d JSON values, not one", values)
	}
	return nil
}

// invalidAt returns the offset of the first byte of data that starts no
// UTF-8 sequence. data is not valid UTF-8, so there is such a byte.
func invalidAt(data []byte) int {
	for i := 0; ; {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			return i
		}
		i += size
	}
}

// ended marks the end of a value inside the innermost of open, and returns
// open: an object's next token is then a name or its end.
func ended(open []*container) []*container {
	if n := len(open); n > 0 && open[n-1].names != nil {
		open[n-1].atName = true
	}
	return open
}

// entryOf returns the entry that the fields of an entry of Format state,
// without its choices, and its token. It returns an error naming the first
// field that is missing, extra or out of its form.
func entryOf(root map[string]json.RawMessage) (Entry, string, error) {
	for _, field := range fields {
		if _, ok := root[field]; !ok {
			return Entry{}, "", fmt.Errorf("the entry lacks %s", field)
		}
	}
	if len(root) != len(fields) {
		return Entry{}, "", fmt.Errorf("the entry states %d fields, not the %d of format %d",
			len(root), len(fields), Format)
	}
	var e Entry
	var ok bool
	if e.Definition, ok = text(root[fieldDefinition]); !ok || !version.MatchString(e.Definition) {
		return Entry{}, "", fmt.Errorf("definition %s is no version", root[fieldDefinition])
	}
	if e.Property, ok = text(root[fieldProperty]); !ok {
		return Entry{}, "", fmt.Errorf("property %s is no string", root[fieldProperty])
	}
	var err error
	if e.Identity, err = identityOf(root[fieldIdentity]); err != nil {
		return Entry{}, "", err
	}
	tok, ok := text(root[fieldChoices])
	if !ok {
		return Entry{}, "", fmt.Errorf("choices %s is no string", root[fieldChoices])
	}
	if e.Counterexample, err = counterexampleOf(root[fieldCounterexample]); err != nil {
		return Entry{}, "", err
	}
	if e.Found, err = dateOf(root[fieldFound]); err != nil {
		return Entry{}, "", err
	}
	return e, tok, nil
}

// identityOf returns the identity that raw states. raw is an object of one
// of the four shapes, whose line is from 1 to 2^31 - 1 and whose other keys
// are non-empty strings.
func identityOf(raw json.RawMessage) (Identity, error) {
	keys, ok := members(raw)
	if !ok {
		return Identity{}, fmt.Errorf("identity %s is no object", raw)
	}
	var id Identity
	texts := map[string]*string{
		keyAssertion: &id.Assertion,
		keyContract:  &id.Contract,
		keyError:     &id.Error,
		keyFile:      &id.File,
	}
	for key, value := range keys {
		if key == keyLine {
			line, err := strconv.Atoi(string(value))
			if !positive.Match(value) || err != nil || line > maxLine {
				return Identity{}, fmt.Errorf("identity's line %s is no line", value)
			}
			id.Line = line
			continue
		}
		field, known := texts[key]
		if !known {
			return Identity{}, fmt.Errorf("identity states the key %q", key)
		}
		if *field, _ = text(value); *field == "" {
			return Identity{}, fmt.Errorf("identity's %s %s is no non-empty string", key, value)
		}
	}
	if !id.Valid() {
		return Identity{}, fmt.Errorf("identity %s has no shape of the format", raw)
	}
	return id, nil
}

// counterexampleOf returns the draws that raw states: an array of objects,
// each with a string label and, when it has one, a value.
func counterexampleOf(raw json.RawMessage) ([]Draw, error) {
	items, ok := elements(raw)
	if !ok {
		return nil, fmt.Errorf("counterexample %s is no array", raw)
	}
	draws := make([]Draw, 0, len(items))
	for i, item := range items {
		keys, ok := members(item)
		if !ok {
			return nil, fmt.Errorf("draw %d %s is no object", i, item)
		}
		label, ok := text(keys[keyLabel])
		if !ok {
			return nil, fmt.Errorf("draw %d %s has no string label", i, item)
		}
		value := keys[keyValue]
		delete(keys, keyLabel)
		delete(keys, keyValue)
		if len(keys) > 0 {
			return nil, fmt.Errorf("draw %d %s states a key other than label and value", i, item)
		}
		draws = append(draws, Draw{Label: label, Value: value})
	}
	return draws, nil
}

// dateOf returns the date that raw states as YYYY-MM-DD, a day of the
// years 1 to 9999.
func dateOf(raw json.RawMessage) (time.Time, error) {
	found, _ := text(raw)
	date, err := time.Parse(time.DateOnly, found)
	if err != nil || date.Year() < 1 {
		return time.Time{}, fmt.Errorf("found %s is no date", raw)
	}
	return date, nil
}

// members returns the members of the JSON object raw, and false when raw is
// no object.
func members(raw []byte) (map[string]json.RawMessage, bool) {
	var m map[string]json.RawMessage
	err := json.Unmarshal(raw, &m)
	return m, err == nil && m != nil
}

// elements returns the elements of the JSON array raw, and false when raw
// is no array.
func elements(raw json.RawMessage) ([]json.RawMessage, bool) {
	var s []json.RawMessage
	err := json.Unmarshal(raw, &s)
	return s, err == nil && s != nil
}

// text returns the string that raw states, and false when raw is no JSON
// string.
func text(raw json.RawMessage) (string, bool) {
	var s *string
	if err := json.Unmarshal(raw, &s); err != nil || s == nil {
		return "", false
	}
	return *s, true
}
