// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"io/fs"

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
	// Expect is pass or fail.
	Expect string `json:"expect"`
	// Detail is what the failure's record must hold, keyed by the
	// names the assertion declares. Every field stated must match; a
	// field the case leaves out is not checked.
	Detail map[string]json.RawMessage `json:"detail"`
	// Subject names the behaviour this case hands the assertion in
	// place of arguments, and is empty for a case that states values.
	Subject struct {
		Kind string `json:"kind"`
	} `json:"subject"`
	// Skip names languages this case does not apply to, and why.
	Skip map[string]string `json:"skip"`
}

// SkipReason returns why this case does not apply to Go, and whether
// it says so.
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

// Check reports how the seat's outcome differs from what the case
// required, and nil when it matches.
//
// It returns an error rather than failing a test so the rule can be
// driven against cases it must reject, the same reason the shared
// suites state their verdict as a value.
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
			return fmt.Errorf("conformance: %s reported no record; "+
				"the assertion did not report one", c.ID)
		}
		return c.checkDetail(records[0])

	default:
		return fmt.Errorf("conformance: %s states an unknown expectation %q", c.ID, c.Expect)
	}
}

// checkDetail reports how a record's detail differs from what the case
// states, and nil when every stated field matches.
//
// A case states values as typed literals, so an int and a float that
// render alike stay apart. Comparison is on the decoded value, because
// what the assertion reports is a Go value rather than a literal.
func (c Case) checkDetail(f assert.Failure) error {
	for name, raw := range c.Detail {
		want, err := Decode(raw)
		if err != nil {
			return fmt.Errorf("conformance: %s detail %q: %w", c.ID, name, err)
		}
		held, ok := f.Detail[name]
		if !ok {
			return fmt.Errorf("conformance: %s record holds no detail %q, want %+v",
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
func Cases() (map[ID][]Case, error) {
	names, err := fs.Glob(definition, corpusGlob)
	if err != nil {
		return nil, fmt.Errorf("conformance: glob the corpus: %w", err)
	}

	out := make(map[ID][]Case, len(names))
	for _, name := range names {
		raw, err := definition.ReadFile(name)
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
		out[file.Assertion] = append(out[file.Assertion], file.Cases...)
	}
	return out, nil
}
