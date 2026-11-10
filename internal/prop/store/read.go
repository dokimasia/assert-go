// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/token"
)

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
	// FieldCounterexample is the list of the draw entries and the step
	// entries of the case, at which the path of a fault in the counterexample
	// starts.
	FieldCounterexample = "counterexample"
	// fieldFound is the date on which the run wrote the entry.
	fieldFound = "found"
)

// The names of the keys of an identity, of a draw and of a step.
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
	// keyStep is a step's action.
	keyStep = "step"
	// keyClient is the client of a step of a concurrent section.
	keyClient = "client"
	// keyDrain is the mark of a step of the drain.
	keyDrain = "drain"
)

// drainMark is the one value of a step's drain mark.
const drainMark = "true"

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

// fields are the fields of an entry of either format, in the order that the
// definition lists them.
var fields = []string{
	fieldStore, fieldDefinition, fieldProperty, fieldIdentity, fieldChoices, FieldCounterexample, fieldFound,
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
//
// The checks run in the definition's order: the file's JSON, the format,
// the fields and their forms, and then the token's version and its
// choices. A file of a later format is skipped whatever its other fields
// state.
//
// # Errors
//
// Read returns a fault of the kind [ErrLater] for [Skip], and one of the
// kind [ErrDamaged] for [Damaged]. The path of a fault in a field leads to
// it through the entry's JSON, as identity.line or counterexample[2].
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

// decode returns the entry that data states. It returns a fault of the
// kind ErrLater for an entry of a later format or token version, and one of
// the kind ErrDamaged when data is no entry of Format or StepsFormat.
func decode(data []byte) (Entry, error) {
	if err := wellFormed(data); err != nil {
		return Entry{}, err
	}
	root, ok := members(data)
	if !ok {
		return Entry{}, damaged("the file is not one JSON object")
	}
	raw := root[fieldStore]
	if !positive.Match(raw) {
		return Entry{}, fault.At(damaged("the entry states no format"), fault.Field(fieldStore))
	}
	// The form of a positive integer leaves Atoi one error, for a format
	// beyond 64 bits, and Atoi then returns the largest int.
	format, _ := strconv.Atoi(string(raw))
	if format > StepsFormat {
		return Entry{}, fault.At(fault.Of(ErrLater, "the entry is of format %s", raw), fault.Field(fieldStore))
	}
	e, tok, err := entryOf(root, format)
	if err != nil {
		return Entry{}, err
	}
	stated := tokenVersion.FindStringSubmatch(tok)
	if stated == nil {
		return Entry{}, fault.At(damaged("%q states no token version", tok), fault.Field(fieldChoices))
	}
	if stated[1] != currentToken {
		return Entry{}, fault.At(fault.Of(ErrLater, "the token is of version %s", stated[1]),
			fault.Field(fieldChoices))
	}
	if e.Choices, err = token.Decode(tok); err != nil {
		return Entry{}, fault.At(damaged("the choices are no token").Because(err), fault.Field(fieldChoices))
	}
	return e, nil
}

// damaged returns a fault of the kind ErrDamaged whose reason is format
// with args, as fmt.Sprintf formats them.
func damaged(format string, args ...any) *fault.Error {
	return fault.Of(ErrDamaged, format, args...)
}

// wellFormed returns why data is no JSON value that every reader reads
// alike, as a fault of the kind ErrDamaged, and nil when it is one: UTF-8
// text of one value, no name that repeats within an object, and at most
// maxDepth levels of objects and arrays.
func wellFormed(data []byte) error {
	if !utf8.Valid(data) {
		return damaged("the file is not UTF-8 from byte %d", invalidAt(data))
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
			return damaged("the file is no JSON").Because(err)
		}
		if n := len(open); n > 0 && open[n-1].atName {
			if tok == closeObject {
				open = ended(open[:n-1])
				continue
			}
			name, _ := tok.(string)
			if _, seen := open[n-1].names[name]; seen {
				return damaged("an object repeats the name %q", name)
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
			return damaged("the file nests past %d levels", maxDepth)
		}
	}
	if values != 1 {
		return damaged("the file states %d JSON values, not one", values)
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

// entryOf returns the entry that the fields of an entry of format state,
// without its choices, and its token. It returns a fault of the kind
// ErrDamaged at the first field that is missing or out of its form, and
// one for an entry of a field too many.
func entryOf(root map[string]json.RawMessage, format int) (Entry, string, error) {
	for _, field := range fields {
		if _, ok := root[field]; !ok {
			return Entry{}, "", fault.At(damaged("the entry lacks the field"), fault.Field(field))
		}
	}
	if len(root) != len(fields) {
		return Entry{}, "", damaged("the entry states %d fields, not the %d of format %d",
			len(root), len(fields), format)
	}
	var e Entry
	var ok bool
	if e.Definition, ok = text(root[fieldDefinition]); !ok || !version.MatchString(e.Definition) {
		return Entry{}, "", fault.At(damaged("%s is no version", stated(root[fieldDefinition])),
			fault.Field(fieldDefinition))
	}
	if e.Property, ok = text(root[fieldProperty]); !ok {
		return Entry{}, "", fault.At(damaged("%s is no string", stated(root[fieldProperty])),
			fault.Field(fieldProperty))
	}
	var err error
	if e.Identity, err = identityOf(root[fieldIdentity]); err != nil {
		return Entry{}, "", fault.At(err, fault.Field(fieldIdentity))
	}
	tok, ok := text(root[fieldChoices])
	if !ok {
		return Entry{}, "", fault.At(damaged("%s is no string", stated(root[fieldChoices])),
			fault.Field(fieldChoices))
	}
	if e.Counterexample, e.Steps, err = counterexampleOf(root[FieldCounterexample], format); err != nil {
		return Entry{}, "", fault.At(err, fault.Field(FieldCounterexample))
	}
	if e.Found, err = dateOf(root[fieldFound]); err != nil {
		return Entry{}, "", fault.At(err, fault.Field(fieldFound))
	}
	return e, tok, nil
}

// identityOf returns the identity that raw states. raw is an object of one
// of the four shapes, whose line is from 1 to 2^31 - 1 and whose other keys
// are non-empty strings.
func identityOf(raw json.RawMessage) (Identity, error) {
	keys, ok := members(raw)
	if !ok {
		return Identity{}, damaged("%s is no object", stated(raw))
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
				return Identity{}, fault.At(damaged("%s is no line", stated(value)), fault.Field(keyLine))
			}
			id.Line = line
			continue
		}
		field, known := texts[key]
		if !known {
			return Identity{}, fault.At(damaged("the key is no key of an identity"), fault.Field(key))
		}
		if *field, _ = text(value); *field == "" {
			return Identity{}, fault.At(damaged("%s is no non-empty string", stated(value)), fault.Field(key))
		}
	}
	if !id.Valid() {
		return Identity{}, damaged("the keys form none of the four shapes of an identity")
	}
	return id, nil
}

// counterexampleOf returns the draws and the steps that raw states in an
// entry of format: an array of objects. A draw entry has a string label and,
// when it has one, a value. An entry of StepsFormat also has step entries,
// which state an action under "step".
func counterexampleOf(raw json.RawMessage, format int) ([]Draw, []Step, error) {
	items, ok := elements(raw)
	if !ok {
		return nil, nil, damaged("%s is no array", stated(raw))
	}
	draws := make([]Draw, 0, len(items))
	var steps []Step
	for i, item := range items {
		keys, ok := members(item)
		if !ok {
			return nil, nil, fault.At(damaged("%s is no object", stated(item)), fault.Index(i))
		}
		if _, step := keys[keyStep]; step {
			if format != StepsFormat {
				return nil, nil, fault.At(damaged("a step entry needs format %d", StepsFormat), fault.Index(i))
			}
			s, err := stepOf(keys)
			if err != nil {
				return nil, nil, fault.At(err, fault.Index(i))
			}
			s.Draws = len(draws)
			steps = append(steps, s)
			continue
		}
		label, ok := text(keys[keyLabel])
		if !ok {
			return nil, nil, fault.At(damaged("the draw states no string label"), fault.Index(i))
		}
		value := keys[keyValue]
		delete(keys, keyLabel)
		delete(keys, keyValue)
		if len(keys) > 0 {
			return nil, nil, fault.At(damaged("the draw states a key other than label and value"), fault.Index(i))
		}
		draws = append(draws, Draw{Label: label, Value: value})
	}
	return draws, steps, nil
}

// stepOf returns the step that the keys of a step entry state: a string
// action, with a client from 0 to 2^31 - 1 or with the drain mark true. A
// step entry has no other key, and never both a client and the drain mark.
func stepOf(keys map[string]json.RawMessage) (Step, error) {
	action, ok := text(keys[keyStep])
	if !ok {
		return Step{}, fault.At(damaged("%s is no string", stated(keys[keyStep])), fault.Field(keyStep))
	}
	s := Step{Action: action, Client: -1}
	client, hasClient := keys[keyClient]
	drain, hasDrain := keys[keyDrain]
	if hasClient && hasDrain {
		return Step{}, damaged("the step entry has both a client and a drain mark")
	}
	want := 1
	if hasClient || hasDrain {
		want = 2
	}
	if len(keys) != want {
		return Step{}, damaged("the step entry has a key other than step, client and drain")
	}
	if hasClient {
		n, err := strconv.Atoi(string(client))
		if err != nil || n < 0 || n > maxClient {
			return Step{}, fault.At(damaged("%s is no client", stated(client)), fault.Field(keyClient))
		}
		s.Client = n
	}
	if hasDrain && string(drain) != drainMark {
		return Step{}, fault.At(damaged("%s is no drain mark", stated(drain)), fault.Field(keyDrain))
	}
	s.Drain = hasDrain
	return s, nil
}

// dateOf returns the date that raw states as YYYY-MM-DD, a day of the
// years 1 to 9999.
func dateOf(raw json.RawMessage) (time.Time, error) {
	found, _ := text(raw)
	date, err := time.Parse(time.DateOnly, found)
	if err != nil || date.Year() < 1 {
		return time.Time{}, damaged("%s is no date", stated(raw))
	}
	return date, nil
}

// stated returns the text that a reason gives for raw, a JSON value. It
// returns the text of a scalar, and "the value" for an object or an array,
// whose text can span lines.
func stated(raw json.RawMessage) string {
	if raw[0] == '{' || raw[0] == '[' {
		return "the value"
	}
	return string(raw)
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
