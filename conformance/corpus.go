// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"io/fs"
	"maps"
	"reflect"
	"slices"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
)

// corpusGlob matches every corpus file in the vendored definition.
const corpusGlob = "spec/corpus/*.json"

// goLanguageKey names this language in a case's skip table.
const goLanguageKey = "go"

// The outcomes a case states.
const (
	expectPass = "pass"
	expectFail = "fail"
)

// Case is one corpus case: what an assertion is given, and what it
// must report.
type Case struct {
	// ID names the case, qualified with its assertion.
	ID string `json:"id"`
	// Assertion is the canonical id the case covers, taken from the
	// file it was read from rather than stated per case.
	Assertion string `json:"-"`
	// Args are the assertion's arguments after the seat, as typed
	// literals, excluding the trailing message.
	Args []json.RawMessage `json:"args"`
	// Options are the ids of the relaxations that the call passes.
	Options []ID `json:"options"`
	// Expect is pass or fail.
	Expect string `json:"expect"`
	// Detail is the detail that the failure's record states, keyed by the
	// names that the assertion declares. Every stated field must match,
	// and a field that the case leaves out is not checked.
	Detail map[string]json.RawMessage `json:"detail"`
	// Fields are the detail fields that the assertion declares, taken
	// from the assertion table and not stated per case. A failure's
	// record contains exactly these.
	Fields []string `json:"-"`
	// Subject names the behaviour that the assertion takes in place of
	// arguments, and is empty for a case that states values.
	Subject struct {
		Kind string `json:"kind"`
	} `json:"subject"`
	// Skip names languages this case does not apply to, and why.
	Skip map[string]string `json:"skip"`
}

// SkipReason returns why this case does not apply to Go, and whether the
// case states a reason.
func (c Case) SkipReason() (string, bool) {
	why, stated := c.Skip[goLanguageKey]
	return why, stated
}

// Decoded materializes the case's arguments as native values.
//
// # Errors
//
// It returns the fault of an argument that is no typed literal, at the
// case's ID, args and the argument's index.
func (c Case) Decoded() ([]any, error) {
	out := make([]any, len(c.Args))
	for i, raw := range c.Args {
		value, err := literal.Decode(raw)
		if err != nil {
			return nil, fault.At(err, fault.Field(c.ID), fault.Field(argsMember), fault.Index(i))
		}
		out[i] = value
	}
	return out, nil
}

// Check returns how the seat's outcome differs from the one that the case
// requires, or nil when they match. aborting states whether the runner
// called the assertion on the aborting surface.
//
// The runner passes the case's ID as the assertion's message. The first
// record of a failing case names the case's assertion, states the ID as
// its contract, contains exactly the case's Fields, and has the case's
// value for every field that the case states.
//
// The recorder keeps the case's one call record. The record is of the
// definition that [Version] reports, has the seq 1 without a parent, and
// states the case's assertion, the ID as its contract, the verdict that the
// case expects, and aborting. A failing case's call record states the
// case's Fields as typed literals, with the case's value for every field
// that the case states. A passing case's call record states no detail.
//
// It returns an error instead of failing a test, so that a test can drive
// the rule with cases that it must refuse. The shared suites state their
// verdict as a value for the same reason.
//
// # Errors
//
// It returns a fault whose path starts at the case's ID, and leads to the
// part of the case that the outcome differs from: expect, or a field of
// detail.
func (c Case) Check(r *assert.Recorder, aborting bool) error {
	switch c.Expect {
	case expectPass:
		if r.Failed() {
			return c.differs(fault.New("the assertion fails: %s", r.Message()), expectMember)
		}

	case expectFail:
		if !r.Failed() {
			return c.differs(fault.New("the assertion passes"), expectMember)
		}
		records := r.Failures()
		if len(records) == 0 {
			return c.differs(fault.New("the assertion fails without a record"))
		}
		if err := c.checkRecord(records[0]); err != nil {
			return err
		}

	default:
		return c.differs(fault.New("%q is neither pass nor fail", c.Expect), expectMember)
	}
	return c.checkCall(callsOf(r), aborting)
}

// checkCall returns how the call records that a recorder keeps differ from
// the case's one call record, or nil when they match. A value of the
// detail matches when it has the canonical text of the value that the case
// states, as a decoding vector's value does.
func (c Case) checkCall(calls []call, aborting bool) error {
	if len(calls) != 1 {
		return c.differs(fault.New("the recorder keeps %d call records, want 1", len(calls)))
	}
	got, detail := calls[0], calls[0].Detail
	got.Detail = nil
	want := call{
		Definition: Version(), Seq: 1, Assertion: c.Assertion, Contract: c.ID, Verdict: c.Expect, Aborting: aborting,
	}
	if !reflect.DeepEqual(got, want) {
		return c.differs(fault.New("the call record is %s, want %s", jsonOf(got), jsonOf(want)))
	}
	if c.Expect == expectPass {
		if detail != nil {
			return c.differs(fault.New("the call record of a pass states the detail %s", detail), detailMember)
		}
		return nil
	}
	var fields map[string]json.RawMessage
	// A call record states its detail as a JSON object.
	_ = json.Unmarshal(detail, &fields)
	stated := slices.Sorted(maps.Keys(fields))
	declared := slices.Sorted(slices.Values(c.Fields))
	if !slices.Equal(stated, declared) {
		return c.differs(fault.New("the call record states the fields %q, want %q", stated, declared), detailMember)
	}
	for _, name := range slices.Sorted(maps.Keys(c.Detail)) {
		path := []fault.Segment{fault.Field(c.ID), fault.Field(detailMember), fault.Key(name)}
		value, err := literal.Decode(fields[name])
		if err != nil {
			return fault.At(fault.New("the call record states no typed literal of the field").Because(err), path...)
		}
		// checkRecord has decoded the value that the case states.
		if same, _ := sameValue(value, c.Detail[name]); !same {
			return fault.At(fault.New("the call record states %s, want %s", fields[name], c.Detail[name]), path...)
		}
	}
	return nil
}

// checkRecord returns how a record differs from the one that the case
// states, or nil when it matches.
//
// A case states values as typed literals, so an int and a float of the
// same rendering differ. The comparison is of the decoded value, because
// the assertion reports a Go value and not a literal.
func (c Case) checkRecord(f assert.Failure) error {
	if f.Assertion != c.Assertion {
		return c.differs(fault.New("the record is of the assertion %q, want %q", f.Assertion, c.Assertion))
	}
	if f.Contract != c.ID {
		return c.differs(fault.New("the record states the contract %q, want the message %q", f.Contract, c.ID))
	}
	reported := slices.Sorted(maps.Keys(f.Detail))
	declared := slices.Sorted(slices.Values(c.Fields))
	if !slices.Equal(reported, declared) {
		return c.differs(fault.New("the record states the fields %q, want %q", reported, declared), detailMember)
	}

	for name, raw := range c.Detail {
		want, err := literal.Decode(raw)
		if err != nil {
			return fault.At(err, fault.Field(c.ID), fault.Field(detailMember), fault.Key(name))
		}
		held, ok := f.Detail[name]
		if !ok {
			return fault.At(fault.New("the record states no such field, want %+v", want),
				fault.Field(c.ID), fault.Field(detailMember), fault.Key(name))
		}
		if !cmp.Equal(held, want, cmpopts.EquateNaNs()) {
			return fault.At(fault.New("the field is %+v, want %+v", held, want),
				fault.Field(c.ID), fault.Field(detailMember), fault.Key(name))
		}
	}
	return nil
}

// differs returns f at the case's ID and at the members of the case that
// the outcome differs from.
func (c Case) differs(f *fault.Error, members ...string) error {
	path := make([]fault.Segment, 0, 1+len(members))
	path = append(path, fault.Field(c.ID))
	for _, m := range members {
		path = append(path, fault.Field(m))
	}
	return fault.At(f, path...)
}

// Cases returns every corpus case, keyed by the assertion it covers. Each
// case states the assertion of its file and the detail fields that the
// assertion table declares for it.
func Cases() map[ID][]Case {
	assertions := Assertions()
	// The pattern is well formed, so Glob returns no error.
	names, _ := fs.Glob(definition, corpusGlob)

	out := make(map[ID][]Case, len(names))
	for _, name := range names {
		var file struct {
			Assertion ID     `json:"assertion"`
			Cases     []Case `json:"cases"`
		}
		read(name, &file)
		for i := range file.Cases {
			file.Cases[i].Assertion = string(file.Assertion)
			file.Cases[i].Fields = assertions[file.Assertion].DetailFields
		}
		out[file.Assertion] = append(out[file.Assertion], file.Cases...)
	}
	return out
}
