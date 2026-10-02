// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"slices"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"go.dokimi.dev/assert"
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
func (c Case) Decoded() ([]any, error) {
	out := make([]any, len(c.Args))
	for i, raw := range c.Args {
		value, err := Decode(raw)
		if err != nil {
			return nil, fmt.Errorf("conformance: %s argument %d: %w", c.ID, i, err)
		}
		out[i] = value
	}
	return out, nil
}

// Check returns how the seat's outcome differs from the one that the case
// requires, or nil when they match.
//
// The runner passes the case's ID as the assertion's message. The first
// record of a failing case names the case's assertion, states the ID as
// its contract, contains exactly the case's Fields, and has the case's
// value for every field that the case states.
//
// It returns an error instead of failing a test, so that a test can drive
// the rule with cases that it must refuse. The shared suites state their
// verdict as a value for the same reason.
func (c Case) Check(r *assert.Recorder) error {
	switch c.Expect {
	case expectPass:
		if r.Failed() {
			return fmt.Errorf("conformance: %s expects pass, got failure: %s", c.ID, r.Message())
		}
		return nil

	case expectFail:
		if !r.Failed() {
			return fmt.Errorf("conformance: %s expects fail, got pass", c.ID)
		}

		records := r.Failures()
		if len(records) == 0 {
			return fmt.Errorf("conformance: %s reported a failure without a record", c.ID)
		}
		return c.checkRecord(records[0])

	default:
		return fmt.Errorf("conformance: %s states an unknown expectation %q", c.ID, c.Expect)
	}
}

// checkRecord returns how a record differs from the one that the case
// states, or nil when it matches.
//
// A case states values as typed literals, so an int and a float of the
// same rendering differ. The comparison is of the decoded value, because
// the assertion reports a Go value and not a literal.
func (c Case) checkRecord(f assert.Failure) error {
	if f.Assertion != c.Assertion {
		return fmt.Errorf("conformance: %s reported a record of %q, want %q", c.ID, f.Assertion, c.Assertion)
	}
	if f.Contract != c.ID {
		return fmt.Errorf("conformance: %s reported the contract %q, want the message %q", c.ID, f.Contract, c.ID)
	}
	reported := slices.Sorted(maps.Keys(f.Detail))
	declared := slices.Sorted(slices.Values(c.Fields))
	if !slices.Equal(reported, declared) {
		return fmt.Errorf("conformance: %s reported the detail fields %q, want the declared %q",
			c.ID, reported, declared)
	}

	for name, raw := range c.Detail {
		want, err := Decode(raw)
		if err != nil {
			return fmt.Errorf("conformance: %s detail %q: %w", c.ID, name, err)
		}
		held, ok := f.Detail[name]
		if !ok {
			return fmt.Errorf("conformance: %s record states no detail %q, want %+v",
				c.ID, name, want)
		}
		if !cmp.Equal(held, want, cmpopts.EquateNaNs()) {
			return fmt.Errorf("conformance: %s detail %q is %+v, want %+v",
				c.ID, name, held, want)
		}
	}
	return nil
}

// Cases returns every corpus case, keyed by the assertion it covers.
func Cases() (map[ID][]Case, error) { return casesIn(definition, corpusGlob) }

// casesIn returns the cases of the corpus files of fsys that glob
// matches, keyed by the assertion that each covers.
func casesIn(fsys fs.FS, glob string) (map[ID][]Case, error) {
	assertions, err := assertionsIn(fsys)
	if err != nil {
		return nil, err
	}
	names, err := fs.Glob(fsys, glob)
	if err != nil {
		return nil, fmt.Errorf("conformance: glob the corpus: %w", err)
	}

	out := make(map[ID][]Case, len(names))
	for _, name := range names {
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("conformance: read %s: %w", name, err)
		}

		var file struct {
			Assertion ID     `json:"assertion"`
			Cases     []Case `json:"cases"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			return nil, fmt.Errorf("conformance: parse %s: %w", name, err)
		}
		declared, ok := assertions[file.Assertion]
		if !ok {
			return nil, fmt.Errorf("conformance: %s covers %q, which the definition does not state",
				name, file.Assertion)
		}
		for i := range file.Cases {
			file.Cases[i].Assertion = string(file.Assertion)
			file.Cases[i].Fields = declared.DetailFields
		}
		out[file.Assertion] = append(out[file.Assertion], file.Cases...)
	}
	return out, nil
}
