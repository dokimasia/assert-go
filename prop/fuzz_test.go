// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/prop"
)

// The environment of a child process that runs FuzzChild.
const (
	// childMode names what FuzzChild runs, and is empty outside a child.
	childMode = "PROP_TEST_FUZZ_MODE"
	// childStore is the directory of the store that FuzzChild uses.
	childStore = "PROP_TEST_FUZZ_STORE"
	// childTimeout bounds the run of a child process.
	childTimeout = time.Minute
)

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
)

// TestFuzz checks what Fuzz reports for a failing input and for the store
// of its fuzz test. Fuzz takes a *testing.F, which only the testing
// package constructs, so each case runs FuzzChild in a child process of
// the test binary. The children run one at a time, because two children
// of a coverage run that exit in the same nanosecond write one coverage
// file.
func TestFuzz(t *testing.T) {
	t.Run("Fuzz", func(t *testing.T) {
		t.Run("reports the shrunk case of a failing input through the input's test", func(t *testing.T) {
			dir := t.TempDir()
			out, err := child(t, failingMode, dir)
			assert.HasError(t, err, "the child fails")
			assert.ContainsInOrder(t, out, []string{
				"the property holds: counterexample after 0 valid and 0 rejected cases, seed ",
				"value: []byte{0x64}",
				"failure of big: it fits",
				"replay: prop.Replay(",
			}, "the record of the smallest failing byte string")
			assert.Length(t, loaded(t, dir).Entries, 1, "the entry of the shrunk case")
		})

		t.Run("fails at once for a stored case that fails", func(t *testing.T) {
			dir := t.TempDir()
			literal, _ := store.Literal([]byte{200})
			save(t, dir, store.Entry{
				Definition:     "1.2.0",
				Property:       contract,
				Identity:       store.Identity{Assertion: big, Contract: fits},
				Choices:        []choice.Choice{sequence(200)},
				Counterexample: []store.Draw{{Label: drawn, Value: literal}},
				Found:          earlier,
			})
			out, err := child(t, storedMode, dir)
			assert.HasError(t, err, "the child fails")
			assert.ContainsInOrder(t, out, []string{
				"the property holds: counterexample after 0 valid and 0 rejected cases, seed ",
				"value: []byte{0xc8}",
			}, "the record of the stored case, as found")
		})

		t.Run("fails at once for a damaged file in the store", func(t *testing.T) {
			dir := t.TempDir()
			write(t, filepath.Join(dir, "damaged.json"), "{")
			out, err := child(t, passingMode, dir)
			assert.HasError(t, err, "the child fails")
			assert.Contains(t, out, "store: read damaged.json: store: not an entry", "the message names the file")
		})

		t.Run("logs a note on a file of a later format", func(t *testing.T) {
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			out, err := child(t, passingMode, dir)
			assert.NoError(t, err, "the child passes")
			assert.Contains(t, out, "later.json: store: later than this reader", "the note names the file")
		})

		t.Run("fails a second property of the fuzz test with the same contract and store", func(t *testing.T) {
			out, err := child(t, twiceMode, t.TempDir())
			assert.HasError(t, err, "the child fails")
			assert.Contains(t, out, duplicateMessage, "the second property is a problem of the fuzz test")
		})

		t.Run("fails at once for a profile other than default and ci", func(t *testing.T) {
			out, err := child(t, passingMode, t.TempDir(), profileVariable+"=nightly")
			assert.HasError(t, err, "the child fails")
			assert.Contains(t, out, profileMessage, "the message names the profile")
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
	if mode == failingMode {
		f.Add([]byte{1, 0, 200})
	}
	prop.Fuzz(f, contract, firstByteFrom100, stored)
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
//
// Under go test -cover the child writes its coverage counters to the
// directory of GOCOVERDIR, whose files the parent merges into its profile.
// A test binary without -test.gocoverdir writes them to a temporary
// directory that it removes.
func child(t *testing.T, mode, dir string, env ...string) (string, error) {
	t.Helper()
	executable, err := os.Executable()
	assert.NoError(t, err, "the test binary's path")
	ctx, cancel := context.WithTimeout(t.Context(), childTimeout)
	defer cancel()
	args := []string{"-test.run=^FuzzChild$", "-test.v", "-test.timeout=" + childTimeout.String()}
	if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
		args = append(args, "-test.gocoverdir="+coverDir)
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = append(os.Environ(), childMode+"="+mode, childStore+"="+dir,
		seedVariable+"=", profileVariable+"=", replayVariable+"=")
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
