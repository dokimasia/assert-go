// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
)

// Definition is the version of the definition that this module implements,
// which every call record states.
const Definition = "3.1.0"

// Verdict is how a call ended.
type Verdict string

// The verdicts of a call.
const (
	// Pass is a call whose assertion found what the contract requires.
	Pass Verdict = "pass"
	// Fail is a call whose assertion did not, and that reported its failure
	// record.
	Fail Verdict = "fail"
	// Error is a call that ended without a verdict, because the library
	// refused an argument or the environment.
	Error Verdict = "error"
)

// Phase is the kind of a property's case that a call ran in.
type Phase string

// The phases of a property's case.
const (
	// NoPhase is the phase of a call that ran in no property's case.
	NoPhase Phase = ""
	// Example is a case whose values the caller states, through Draws or
	// Example.
	Example Phase = "example"
	// Stored is a case that the store keeps.
	Stored Phase = "stored"
	// Simplest is the case whose every choice is its target.
	Simplest Phase = "simplest"
	// Random is a random case before the first coverage check.
	Random Phase = "random"
	// Prefix is a prefix case.
	Prefix Phase = "prefix"
	// Edge is an edge case.
	Edge Phase = "edge"
	// Coverage is a random case after the first coverage check.
	Coverage Phase = "coverage"
	// Replay is the replay of the failing case before shrinking.
	Replay Phase = "replay"
	// Shrink is a candidate that the shrinker runs.
	Shrink Phase = "shrink"
	// Explain is a filling or a boundary step of the explain phase.
	Explain Phase = "explain"
	// Token is the case of a replay token.
	Token Phase = "token"
	// Fuzz is the case that the fuzz bridge decodes from a fuzzer's input.
	Fuzz Phase = "fuzz"
)

// Where is the site of a call: its file and its line. A record states the
// base name of the file, and leaves out a zero Where, the site of a call
// that could not read its frame.
type Where struct {
	// File is the path of the file.
	File string `json:"file"`
	// Line is the line, from 1.
	Line int `json:"line"`
}

// Call is the record of one assertion call, before the [Calls] that keeps
// it gives it its number, and its parent, run and phase.
type Call struct {
	// Assertion is the canonical id of the assertion.
	Assertion string
	// Contract is the caller's message, unchanged.
	Contract string
	// Verdict is how the call ended.
	Verdict Verdict
	// Aborting reports whether the call was made on the aborting surface.
	Aborting bool
	// Where is the call site.
	Where Where
	// Detail is the JSON object of the call's detail, and nil for a call
	// without one. It must be JSON.
	Detail json.RawMessage
	// Error is the text of the fault that ended a call of the verdict
	// [Error].
	Error string
}

// line is a call record as its JSON states it, with its fields in the
// order that the definition lists them.
type line struct {
	Definition string          `json:"definition"`
	Seq        int             `json:"seq"`
	Parent     int             `json:"parent,omitempty"`
	Run        int             `json:"run,omitempty"`
	Phase      Phase           `json:"phase,omitempty"`
	Assertion  string          `json:"assertion"`
	Contract   string          `json:"contract"`
	Verdict    Verdict         `json:"verdict"`
	Aborting   bool            `json:"aborting"`
	Where      *Where          `json:"where,omitempty"`
	Detail     json.RawMessage `json:"detail,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// encode returns the record of e, which its Calls numbered, as one line of
// JSON.
//
// # Panics
//
// It panics when the call's detail is no JSON, which every caller in this
// module builds as JSON.
func encode(e *entry) string {
	l := line{
		Definition: Definition,
		Seq:        e.seq,
		Assertion:  e.call.Assertion,
		Contract:   e.call.Contract,
		Verdict:    e.call.Verdict,
		Aborting:   e.call.Aborting,
		Detail:     e.call.Detail,
		Error:      e.call.Error,
	}
	if e.parent != nil {
		l.Parent, l.Run, l.Phase = e.parent.seq, e.run, e.phase
	}
	if e.call.Where != (Where{}) {
		l.Where = &Where{File: filepath.Base(e.call.Where.File), Line: e.call.Where.Line}
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(l); err != nil {
		panic(fmt.Sprintf("record: the detail of a call of %s is no JSON: %v", e.call.Assertion, err))
	}
	return string(bytes.TrimSuffix(b.Bytes(), []byte("\n")))
}
