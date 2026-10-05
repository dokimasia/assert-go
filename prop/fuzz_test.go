// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/record"
	"go.dokimi.dev/assert/prop"
)

// fuzzOp is the operation of Fuzz, which names its faults.
const fuzzOp = "prop.Fuzz"

// The environment of a child process that runs FuzzChild.
const (
	// childMode names what FuzzChild runs, and is empty outside a child.
	childMode = "PROP_TEST_FUZZ_MODE"
	// childStore is the directory of the store that FuzzChild uses.
	childStore = "PROP_TEST_FUZZ_STORE"
)

// located matches a record of a run whose line starts with the file and the
// line that testing puts before a log line.
var located = regexp.MustCompile(`\.go:\d+: the property holds`)

// The modes of FuzzChild.
const (
	// failingMode fuzzes firstByteFrom100 from a seed that fails.
	failingMode = "failing"
	// storedMode fuzzes firstByteFrom100 without a seed.
	storedMode = "stored"
	// passingMode fuzzes a body that never fails.
	passingMode = "passing"
	// twiceMode fuzzes a body that never fails, in two properties of one
	// contract.
	twiceMode = "twice"
	// detachedMode fuzzes detached from a seed, a failure that no entry of
	// the store can keep.
	detachedMode = "detached"
	// recordingMode fuzzes passesTrue from a seed.
	recordingMode = "recording"
	// rejectingMode fuzzes a body that rejects every input, from a seed.
	rejectingMode = "rejecting"
)

// TestFuzz checks what Fuzz reports for a failing input and for the store
// of its fuzz test. Fuzz takes a *testing.F, which only the testing
// package constructs, so each case runs FuzzChild in a child process of
// the test binary. The children run one at a time, because two children
// of a coverage run that exit in the same nanosecond write one coverage
// file.
//
// A case reads the call records that a child writes. A case of the text
// that a child writes into a test's log, which no call record states,
// builds the text it expects with the writer.
func TestFuzz(t *testing.T) {
	t.Run("Fuzz", func(t *testing.T) {
		t.Run("reports the shrunk case of a failing input through the input's test", func(t *testing.T) {
			dir := t.TempDir()
			out, err := child(t, failingMode, dir, record.Variable+"=1")
			assert.HasError(t, err, "the child fails")
			call := childCall(t, out, "FuzzChild/seed#0", 1)
			assert.Equal(t, []any{call["contract"], call["verdict"]}, []any{contract, "fail"}, "the input's call fails")
			detail, _ := call["detail"].(map[string]any)
			assert.Equal(t, counts(detail), []any{"counterexample", 0.0, 0.0},
				"a counterexample after no valid and no rejected case")
			lit, _ := literal.Encode([]byte{100})
			assert.Equal(t, drawnOf(detail), [][2]any{{drawn, jsonTree(t, string(lit))}},
				"the smallest failing byte string")
			failure := map[string]any{"assertion": big, "contract": fits, "detail": map[string]any{}}
			assert.Equal(t, detail[failureField], any(failure), "the failure of the shrunk case")
			assert.HasPrefix(t, detail[choicesField], "prop1:", "the token that replays the shrunk case")
			if matcher.Mutated() {
				assert.Empty(t, loaded(t, dir).Entries, unwritten)
				return
			}
			assert.Length(t, loaded(t, dir).Entries, 1, "the entry of the shrunk case")
		})

		t.Run("writes the record of a failing input without a source location", func(t *testing.T) {
			out, err := child(t, failingMode, t.TempDir())
			assert.HasError(t, err, "the child fails")
			assert.Contains(t, out, "the property holds: counterexample", "the record of the failing input")
			assert.False(t, located.MatchString(out), "the record's line starts with no file and line")
		})

		t.Run("logs the fault of a store that cannot keep the case of a failing input", func(t *testing.T) {
			dir := t.TempDir()
			out, err := child(t, detachedMode, dir)
			assert.HasError(t, err, "the child fails")
			unkept := &fault.Error{
				Op:     fuzzOp,
				Path:   fault.Path{fault.Field(dir)},
				Reason: fmt.Sprintf("the store keeps no case of %q", contract),
			}
			if matcher.Mutated() {
				assert.NotContains(t, out, matcher.RenderFault(unkept), "no fault of the store, because "+unwritten)
				return
			}
			assert.Contains(t, out, matcher.RenderFault(unkept), "the fault at the store, before its cause")
		})

		t.Run("fails at once for a stored case that fails", func(t *testing.T) {
			dir := t.TempDir()
			lit, _ := literal.Encode([]byte{200})
			save(t, dir, store.Entry{
				Definition:     "1.2.0",
				Property:       contract,
				Identity:       store.Identity{Assertion: big, Contract: fits},
				Choices:        []choice.Choice{sequence(200)},
				Counterexample: []store.Draw{{Label: drawn, Value: lit}},
				Found:          earlier,
			})
			out, err := child(t, storedMode, dir, record.Variable+"=1")
			assert.HasError(t, err, "the child fails")
			call := childCall(t, out, "FuzzChild", 1)
			detail, _ := call["detail"].(map[string]any)
			assert.Equal(t, []any{call["verdict"], counts(detail)}, []any{"fail", []any{"counterexample", 0.0, 0.0}},
				"the call of the test fails before any valid case")
			assert.Equal(t, drawnOf(detail), [][2]any{{drawn, jsonTree(t, string(lit))}}, "the stored case, as found")
		})

		t.Run("counts the stored cases that pass before a stored case that fails", func(t *testing.T) {
			dir := t.TempDir()
			passing, _ := literal.Encode([]byte{7})
			lit, _ := literal.Encode([]byte{200})
			save(t, dir, store.Entry{
				Definition:     "1.2.0",
				Property:       contract,
				Identity:       store.Identity{Assertion: big, Contract: fits},
				Choices:        []choice.Choice{sequence(7)},
				Counterexample: []store.Draw{{Label: drawn, Value: passing}},
				Found:          earlier,
			})
			save(t, dir, store.Entry{
				Definition:     "1.2.0",
				Property:       contract,
				Identity:       store.Identity{Assertion: big, Contract: fits},
				Choices:        []choice.Choice{sequence(200)},
				Counterexample: []store.Draw{{Label: drawn, Value: lit}},
				Found:          earlier.Add(time.Hour),
			})
			out, err := child(t, storedMode, dir, record.Variable+"=1")
			assert.HasError(t, err, "the child fails")
			detail, _ := childCall(t, out, "FuzzChild", 1)["detail"].(map[string]any)
			assert.Equal(t, counts(detail), []any{"counterexample", 1.0, 0.0},
				"a counterexample after the stored case that passed")
		})

		t.Run("logs the fault of a stored case that decodes to other values than it records", func(t *testing.T) {
			dir := t.TempDir()
			recorded, _ := literal.Encode([]byte{8})
			moved := store.Entry{
				Definition:     "1.2.0",
				Property:       contract,
				Identity:       store.Identity{Assertion: big, Contract: fits},
				Choices:        []choice.Choice{sequence(7)},
				Counterexample: []store.Draw{{Label: drawn, Value: recorded}},
				Found:          earlier,
			}
			save(t, dir, moved)
			out, err := child(t, storedMode, dir)
			assert.NoError(t, err, "the child passes")
			decoded := &fault.Error{
				Op:     fuzzOp,
				Path:   fault.Path{fault.Field(dir), fault.Field(moved.Name())},
				Reason: decodedReason,
			}
			assert.Contains(t, out, matcher.RenderFault(decoded), "the fault at the entry's file")
		})

		t.Run("fails at once for a damaged file in the store", func(t *testing.T) {
			dir := t.TempDir()
			write(t, filepath.Join(dir, "damaged.json"), "{")
			out, err := child(t, passingMode, dir, record.Variable+"=1")
			assert.HasError(t, err, "the child fails")
			damaged := &fault.Error{
				Op:   fuzzOp,
				Path: fault.Path{fault.Field(dir)},
				Err: &fault.Error{
					Path:   fault.Path{fault.Field("damaged.json")},
					Kind:   store.ErrDamaged,
					Reason: "the file is not one JSON object",
				},
			}
			expectEnded(t, childCall(t, out, "FuzzChild", 1), damaged)
		})

		t.Run("logs the fault of a file of a later format", func(t *testing.T) {
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			out, err := child(t, passingMode, dir)
			assert.NoError(t, err, "the child passes")
			later := laterFault(fuzzOp, dir, "later.json", "2")
			assert.Contains(t, out, matcher.RenderFault(&later), "the fault at the file")
		})

		t.Run("fails a second property of the fuzz test with the same contract and store", func(t *testing.T) {
			dir := t.TempDir()
			out, err := child(t, twiceMode, dir, record.Variable+"=1")
			assert.HasError(t, err, "the child fails")
			duplicated := &fault.Error{Op: fuzzOp, Path: fault.Path{fault.Field(dir)}, Reason: duplicateReason}
			expectEnded(t, childCall(t, out, "FuzzChild", 2), duplicated)
		})

		t.Run("fails at once for a profile other than default, ci and campaign", func(t *testing.T) {
			out, err := child(t, passingMode, t.TempDir(), profileVariable+"=nightly", record.Variable+"=1")
			assert.HasError(t, err, "the child fails")
			profile := profileFault(fuzzOp)
			expectEnded(t, childCall(t, out, "FuzzChild", 1), &profile)
		})

		t.Run("reads no budget under the campaign profile, and runs no campaign", func(t *testing.T) {
			_, err := child(t, passingMode, t.TempDir(), profileVariable+"=campaign")
			assert.NoError(t, err, "the child passes without a budget")
		})

		t.Run("records each input as a separate call, with its case's calls under the phase fuzz", func(t *testing.T) {
			out, err := child(t, recordingMode, t.TempDir(), record.Variable+"=1")
			assert.NoError(t, err, "the child passes")
			assert.ContainsInOrder(t, out, []string{
				"=== ATTR  FuzzChild/seed#0 dokimi.assert.2 ",
				"=== ATTR  FuzzChild/seed#0 dokimi.assert.1 ",
			}, "the call of the input's case, then the input's call, which ends after it")
			inCase, input := childCall(t, out, "FuzzChild/seed#0", 2), childCall(t, out, "FuzzChild/seed#0", 1)
			assert.Equal(t, []any{inCase["parent"], inCase["run"], inCase["phase"], inCase["assertion"]},
				[]any{1.0, 1.0, "fuzz", "true"}, "the call of the case under the input's call")
			assert.Equal(t, []any{input["assertion"], input["contract"], input["verdict"]},
				[]any{"prop-for-all", contract, "pass"}, "the input's call passes")
		})

		t.Run("records a rejected input as a run of one rejected case", func(t *testing.T) {
			out, err := child(t, rejectingMode, t.TempDir(), record.Variable+"=1")
			assert.NoError(t, err, "the child passes")
			detail, _ := childCall(t, out, "FuzzChild/seed#0", 1)["detail"].(map[string]any)
			assert.Equal(t, []any{detail[casesField], detail[rejectedField]}, []any{0.0, 1.0},
				"the detail of the input's run")
		})

		t.Run("records the stored cases of the fuzz test under the call of the test", func(t *testing.T) {
			dir := t.TempDir()
			lit, _ := literal.Encode([]byte{7})
			save(t, dir, store.Entry{
				Definition:     "1.2.0",
				Property:       contract,
				Identity:       store.Identity{Assertion: big, Contract: fits},
				Choices:        []choice.Choice{sequence(7)},
				Counterexample: []store.Draw{{Label: drawn, Value: lit}},
				Found:          earlier,
			})
			out, err := child(t, recordingMode, dir, record.Variable+"=1")
			assert.NoError(t, err, "the child passes")
			assert.ContainsInOrder(t, out, []string{
				"=== ATTR  FuzzChild dokimi.assert.2 ",
				"=== ATTR  FuzzChild dokimi.assert.1 ",
			}, "the call of the stored case, then the call of the test")
			stored, test := childCall(t, out, "FuzzChild", 2), childCall(t, out, "FuzzChild", 1)
			assert.Equal(t, []any{stored["parent"], stored["run"], stored["phase"], stored["assertion"]},
				[]any{1.0, 1.0, "stored", "true"}, "the call of the stored case under the call of the test")
			detail, _ := test["detail"].(map[string]any)
			assert.Equal(t, []any{test["verdict"], test["aborting"], detail[casesField], detail[rejectedField]},
				[]any{"pass", true, 1.0, 0.0}, "the call of the test passes and counts the stored case")
		})
	})
}

// FuzzPassing fuzzes a body that never fails, from the seeds of its corpus
// in an ordinary test run.
func FuzzPassing(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{2, 0, 7, 8})
	prop.Fuzz(f, contract, draws(prop.Bytes()))
}

// FuzzChild runs Fuzz in a child process of TestFuzz, in the mode that
// PROP_TEST_FUZZ_MODE names, with the store that PROP_TEST_FUZZ_STORE
// names. It skips outside a child process.
func FuzzChild(f *testing.F) {
	mode := os.Getenv(childMode)
	if mode == "" {
		f.Skip("runs in a child process of TestFuzz")
	}
	stored := prop.Store(os.Getenv(childStore))
	if mode == passingMode {
		prop.Fuzz(f, contract, draws(prop.Bytes()), stored)
		return
	}
	if mode == twiceMode {
		prop.Fuzz(f, contract, draws(prop.Bytes()), stored)
		prop.Fuzz(f, contract, draws(prop.Bytes()), stored)
		return
	}
	if mode == detachedMode {
		f.Add([]byte{3})
		prop.Fuzz(f, contract, detached, stored)
		return
	}
	if mode == recordingMode {
		f.Add([]byte{5})
		prop.Fuzz(f, contract, passesTrue, stored)
		return
	}
	if mode == rejectingMode {
		f.Add([]byte{1})
		prop.Fuzz(f, contract, func(c *prop.Case) {
			c.Draw(prop.Bytes(), drawn)
			c.Assume(false)
		}, stored)
		return
	}
	if mode == failingMode {
		f.Add([]byte{1, 0, 200})
	}
	prop.Fuzz(f, contract, firstByteFrom100, stored)
}

// passesTrue is the body that draws a byte string and passes a call of
// true.
func passesTrue(c *prop.Case) {
	c.Draw(prop.Bytes(), drawn)
	assert.True(c, true, "the input passes")
}

// firstByteFrom100 is the body that draws a byte string, and ends the case
// at a first byte of 100 or more with a record whose identity an entry
// keeps.
func firstByteFrom100(c *prop.Case) {
	v := c.Draw(prop.Bytes(), drawn)
	if len(v) > 0 && v[0] >= 100 {
		c.Report(assert.Failure{Assertion: big, Contract: fits}, true)
	}
}

// child runs FuzzChild in a child process of the test binary in mode, with
// the store dir and the extra environment variables of env, and returns
// the process's output and its error. Each variable of a run's
// environment is empty unless env sets it.
func child(t *testing.T, mode, dir string, env ...string) (string, error) {
	t.Helper()
	vars := []string{
		childMode + "=" + mode, childStore + "=" + dir, seedVariable + "=", profileVariable + "=", replayVariable + "=",
		budgetVariable + "=",
	}
	return childtest.Run(t, "FuzzChild", append(vars, env...)...)
}

// childCall returns the JSON object of the call record numbered seq of the
// test name in out, the output of a child whose calls are recorded.
func childCall(t *testing.T, out, name string, seq int) map[string]any {
	t.Helper()
	pattern := `=== ATTR  ` + regexp.QuoteMeta(name) + ` dokimi\.assert\.` + strconv.Itoa(seq) + ` (.*)`
	line := regexp.MustCompile(pattern).FindStringSubmatch(out)
	assert.Length(t, line, 2, "the output states the call record")
	return decodedCalls(t, line[1:])[0]
}

// expectEnded checks that call, the JSON object of a call record, states a
// call that ended without a verdict because of err, with the writer's text
// of err.
func expectEnded(t *testing.T, call map[string]any, err error) {
	t.Helper()
	assert.Equal(t, []any{call["verdict"], call["error"]}, []any{"error", matcher.RenderFault(err)},
		"a call that ended with the fault")
}

// drawnOf returns the label and the value of each draw of the
// counterexample that detail, the JSON object of a run's detail, states.
func drawnOf(detail map[string]any) [][2]any {
	draws, _ := detail[counterexampleField].([]any)
	out := make([][2]any, len(draws))
	for i, d := range draws {
		draw, _ := d.(map[string]any)
		out[i] = [2]any{draw["label"], draw["value"]}
	}
	return out
}
