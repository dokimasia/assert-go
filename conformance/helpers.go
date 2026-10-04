// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"maps"
	"slices"
	"strconv"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
)

// jsonNull is the JSON text of null, which a vector states for an output
// that a run does not produce.
const jsonNull = "null"

// call is a call record that a recorder keeps, with the fields that a
// runner compares. A field that the record does not state is the zero
// value, because no number of a record is 0.
type call struct {
	Definition string          `json:"definition"`
	Seq        int             `json:"seq"`
	Parent     int             `json:"parent,omitempty"`
	Run        int             `json:"run,omitempty"`
	Phase      string          `json:"phase,omitempty"`
	Assertion  string          `json:"assertion"`
	Contract   string          `json:"contract"`
	Verdict    string          `json:"verdict"`
	Aborting   bool            `json:"aborting"`
	Detail     json.RawMessage `json:"detail,omitempty"`
}

// callsOf returns the call records that r keeps, in the order of their
// numbers.
func callsOf(r *assert.Recorder) []call {
	lines := r.Records()
	out := make([]call, len(lines))
	for i, line := range lines {
		// Records returns the JSON objects that the module encodes.
		_ = json.Unmarshal([]byte(line), &out[i])
	}
	return out
}

// at returns err at the segments segs, outermost first, as fault.At does,
// and nil for a nil err.
func at(err error, segs ...fault.Segment) error {
	if err == nil {
		return nil
	}
	return fault.At(err, segs...)
}

// decode decodes raw, the JSON of a vector, into v. It returns a fault
// whose cause is the error of the decoder for a vector that does not parse.
func decode(raw json.RawMessage, v any) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return fault.New("the vector does not parse").Because(err)
	}
	return nil
}

// parseSeed returns the seed that a vector states in decimal, and a fault
// at the member seed for one that is no decimal number of 64 bits.
func parseSeed(text string) (uint64, error) {
	seed, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, fault.At(fault.New("the seed is no decimal number of 64 bits").Because(err), fault.Field(seedMember))
	}
	return seed, nil
}

// sameValue reports whether got is the value that the typed literal want
// states: the two have one canonical text. It returns the fault of a want
// that is no typed literal.
func sameValue(got any, want json.RawMessage) (bool, error) {
	w, err := literal.Decode(want)
	if err != nil {
		return false, err
	}
	return literal.Canonical(got) == literal.Canonical(w), nil
}

// compareDetail returns how reported, the detail of a failure, differs
// from stated, the fields that a vector states as typed literals, or nil
// when each stated field has the stated value. whose names what reported
// the detail, as "the record". A fault is at the stated field below the
// segments at, in the order of the fields' names.
func compareDetail(
	whose string, stated map[string]json.RawMessage, reported map[string]any, at ...fault.Segment,
) error {
	for _, name := range slices.Sorted(maps.Keys(stated)) {
		path := append(slices.Clip(at), fault.Field(detailMember), fault.Key(name))
		got, ok := reported[name]
		if !ok {
			return fault.At(fault.New("%s states no such field, want %s", whose, stated[name]), path...)
		}
		same, err := sameValue(got, stated[name])
		if err != nil {
			return fault.At(err, path...)
		}
		if !same {
			return fault.At(fault.New("the field is %s, want %s", literal.Canonical(got), stated[name]), path...)
		}
	}
	return nil
}

// jsonOf returns the JSON text of v, for the reason of a fault, and the
// empty string for a value that does not encode.
func jsonOf(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
