// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"encoding/json"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/record"
)

// calls is the Calls that a seat of the test embeds, as a seat of this
// module embeds it.
type calls = record.Calls

// keeping is a seat of the test that keeps the records of its calls, as a
// recorder does.
type keeping struct {
	calls
}

// newKeeping returns a seat that keeps the records of its calls.
func newKeeping() *keeping {
	k := &keeping{}
	record.Keep(&k.calls)
	return k
}

// body is the seat of one run of a body, as an attempt of Eventually is.
type body struct {
	calls
}

// attrs is a test's seat: it keeps every attribute written to it, in order.
type attrs struct {
	// name is the test's name.
	name string
	// mu guards written.
	mu sync.Mutex
	// written are the attributes, each a key and a value.
	written [][2]string
}

// Attr keeps the attribute key with value.
func (a *attrs) Attr(key, value string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.written = append(a.written, [2]string{key, value})
}

// Name returns the test's name.
func (a *attrs) Name() string {
	return a.name
}

// attributes returns the attributes written so far.
func (a *attrs) attributes() [][2]string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([][2]string(nil), a.written...)
}

// call returns a passing record of assertion with the contract.
func call(assertion, contract string) record.Call {
	return record.Call{Assertion: assertion, Contract: contract, Verdict: record.Pass, Aborting: true}
}

// decoded returns the JSON object of a record line, failing the test when
// the line is no JSON object.
func decoded(t *testing.T, line string) map[string]any {
	t.Helper()
	var out map[string]any
	assert.NoError(t, json.Unmarshal([]byte(line), &out), "the record is a JSON object")
	return out
}

// runChild runs the test again in a child process, with the variables of
// env, and returns the child's output. It fails the test when the child
// fails or does not run the test.
func runChild(t *testing.T, env ...string) string {
	t.Helper()
	out, err := childtest.Run(t, t.Name(), env...)
	assert.NoError(t, err, "the child passes:\n"+out)
	assert.Contains(t, out, "--- PASS: "+t.Name()+" ", "the child runs the test")
	return out
}
