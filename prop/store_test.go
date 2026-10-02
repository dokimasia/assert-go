// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/prop"
)

// The messages of a run that the store tests pin.
const (
	// duplicateMessage is the problem of a second property of a test with
	// the contract of the first in the same store.
	duplicateMessage = `prop: two properties of the test have the contract "the property holds", ` +
		"and would share their stored cases"
	// decodedNote is the note on a stored case that decodes to other values
	// than it records, around the name of its entry.
	decodedNote = "prop: the stored case %s decodes to other values than it records, and the run tested those"
)

// The facts of the entries that the store tests write and read.
const (
	// fits is the contract of the record that a test body fails with.
	fits = "it fits"
	// fileMode is the mode of a file that a test writes into a store.
	fileMode = 0o644
)

// The dates of the entries that the store tests write and read.
var (
	// today is the time of the seats' controlled clocks.
	today = time.Date(2026, time.October, 1, 9, 30, 0, 0, time.UTC)
	// earlier is the date of an entry found before today.
	earlier = time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
)

// testSeat is a recorder with a test's name, a log and cleanups, as
// testing.TB has them. Its clock reads today.
type testSeat struct {
	*assert.Recorder
	// name is the test's name.
	name string
	// mu guards logs and cleanups.
	mu sync.Mutex
	// logs are the messages of Logf, in call order.
	logs []string
	// cleanups are the functions that end runs, in the order registered.
	cleanups []func()
}

// newTestSeat returns a seat of the test name whose clock reads today.
func newTestSeat(name string) *testSeat {
	return &testSeat{Recorder: assert.NewRecorder().WithClock(assert.NewControlled(today)), name: name}
}

// Name returns the test's name.
func (s *testSeat) Name() string {
	return s.name
}

// Logf adds the formatted message to the log.
func (s *testSeat) Logf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, fmt.Sprintf(format, args...))
}

// Cleanup registers f to run when the test ends.
func (s *testSeat) Cleanup(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanups = append(s.cleanups, f)
}

// end runs the registered functions, the last first, as a test's end does.
func (s *testSeat) end() {
	s.mu.Lock()
	cleanups := s.cleanups
	s.cleanups = nil
	s.mu.Unlock()
	for _, f := range slices.Backward(cleanups) {
		f()
	}
}

// logged returns a copy of the log.
func (s *testSeat) logged() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.logs)
}

// TestStore checks what a run reads from its store and writes to it, and
// the claim that keeps two properties of a test from sharing one store.
func TestStore(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the minimal failing case of a counterexample", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			prop.ForAll(newTestSeat(t.Name()), contract, failsFrom(1001), prop.Seed(7), prop.Store(dir))
			entries := loaded(t, dir).Entries
			assert.Length(t, entries, 1, "one entry")
			got, want := entries[0], entry(1001)
			version, err := conformance.Version()
			assert.NoError(t, err, "the vendored definition states its version")
			assert.Equal(t, got.Name(), want.Name(), "the name of the contract and the minimal case's token")
			assert.Equal(t, got.Definition, version, "the version of the vendored definition")
			assert.Equal(t, got.Identity, want.Identity, "the failure's identity")
			assert.Equal(t, got.Found, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC), "the seat's date")
			assert.Equal(t, valuesOf(got.Counterexample), valuesOf(want.Counterexample), "the drawn values")
		})

		t.Run("keeps the base name of the file and the line of a failure with a location", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			var at assert.Where
			body := func(c *prop.Case) {
				assert.True(c, c.Draw(prop.Integer(0, 10000), drawn) < 1001, "it fits"+here(&at))
			}
			prop.ForAll(newTestSeat(t.Name()), contract, body, prop.Seed(7), prop.Store(dir))
			want := store.Identity{Assertion: "true", File: "store_test.go", Line: at.Line}
			assert.Equal(t, identities(loaded(t, dir).Entries), []store.Identity{want}, "the assertion and its frame")
		})

		t.Run("keeps the type of a panic as the error of its identity", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			var at assert.Where
			body := func(c *prop.Case) {
				if c.Draw(prop.Integer(0, 10000), drawn) >= 1001 {
					panic(fmt.Errorf("too big%s", here(&at)))
				}
			}
			prop.ForAll(newTestSeat(t.Name()), contract, body, prop.Seed(7), prop.Store(dir))
			want := store.Identity{Error: "*errors.errorString", File: "store_test.go", Line: at.Line}
			assert.Equal(t, identities(loaded(t, dir).Entries), []store.Identity{want}, "the panic's type and frame")
		})

		t.Run("writes an entry for each failure", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			body := func(c *prop.Case) {
				v := c.Draw(prop.Integer(0, 1000), drawn)
				if v%2 == 1 {
					c.Report(assert.Failure{Assertion: "odd", Contract: fits}, true)
				}
				if v >= 51 {
					c.Report(assert.Failure{Assertion: big, Contract: fits}, true)
				}
			}
			prop.ForAll(newTestSeat(t.Name()), contract, body, prop.Seed(7), prop.Store(dir))
			assert.Length(t, loaded(t, dir).Entries, 2, "the odd and the big failure")
		})

		t.Run("tries a stored case before the simplest", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			save(t, dir, entry(950))
			got := detailOfOn(newTestSeat(t.Name()), failsFrom(900), prop.Seed(7), prop.Store(dir))
			assert.Equal(t, got[casesField], any(0), "no valid case before the stored one")
			assert.Equal(t, got[choicesField], any("prop1:AIQH"), "900, the minimal case of the vector")
			assert.Length(t, loaded(t, dir).Entries, 2, "the stored case and the minimal one")
		})

		t.Run("leaves an existing entry of the same name unchanged", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			old := entry(1001)
			save(t, dir, old)
			before := read(t, filepath.Join(dir, old.Name()))
			prop.ForAll(newTestSeat(t.Name()), contract, failsFrom(1001), prop.Seed(7), prop.Store(dir))
			assert.Equal(t, read(t, filepath.Join(dir, old.Name())), before, "the stored file")
		})

		t.Run("logs a note on a file of a later format", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Seed(7), prop.Store(dir))
			assert.False(t, seat.Failed(), "the run passes")
			assert.Length(t, seat.logged(), 1, "one note")
			assert.HasPrefix(t, seat.logged()[0], "later.json: store: later than this reader",
				"the note names the file")
		})

		t.Run("logs a note on a stored case that decodes to other values than it records", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			moved := entry(950)
			moved.Counterexample[0].Value = json.RawMessage(`{"type": "int", "value": 951}`)
			save(t, dir, moved)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, failsFrom(900), prop.Seed(7), prop.Store(dir))
			assert.Equal(t, seat.logged(), []string{fmt.Sprintf(decodedNote, moved.Name())}, "the note names the entry")
		})

		tests := []struct {
			name    string
			changed func(*store.Entry)
		}{
			{
				name: "logs no note on a stored value that another spelling of the same JSON states",
				changed: func(e *store.Entry) {
					e.Counterexample[0].Value = json.RawMessage(`{"value":950,"type":"int"}`)
				},
			},
			{
				name:    "logs no note on a stored draw without a value",
				changed: func(e *store.Entry) { e.Counterexample[0].Value = nil },
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				e := entry(950)
				tt.changed(&e)
				save(t, dir, e)
				seat := newTestSeat(t.Name())
				prop.ForAll(seat, contract, failsFrom(900), prop.Seed(7), prop.Store(dir))
				assert.Empty(t, seat.logged(), "no note")
			})
		}

		differing := []struct {
			name    string
			changed func(*store.Entry)
		}{
			{name: "logs a note on a stored draw of another label", changed: func(e *store.Entry) {
				e.Counterexample[0].Label = "other"
			}},
			{name: "logs a note on another number of stored draws", changed: func(e *store.Entry) {
				e.Counterexample = append(e.Counterexample, store.Draw{Label: drawn})
			}},
		}
		for _, tt := range differing {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				e := entry(950)
				tt.changed(&e)
				write(t, filepath.Join(dir, e.Name()), document(e))
				seat := newTestSeat(t.Name())
				prop.ForAll(seat, contract, failsFrom(900), prop.Seed(7), prop.Store(dir))
				assert.Length(t, seat.logged(), 1, "one note on the decoding")
			})
		}

		t.Run("logs a note on a stored value whose draw now has no typed literal", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			save(t, dir, entry(950))
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, draws(prop.Just(complex(1, 2))), prop.Seed(7), prop.Store(dir))
			assert.Length(t, seat.logged(), 1, "one note on the decoding")
			assert.False(t, seat.Failed(), "the run passes")
		})

		t.Run("logs a note on a store that cannot keep the case", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, detached, prop.Seed(7), prop.Store(dir))
			want := fmt.Sprintf("prop: the store %s keeps no case of %q: ", dir, contract)
			assert.Length(t, seat.logged(), 1, "one note")
			assert.HasPrefix(t, seat.logged()[0], want, "the note names the store")
			assert.Length(t, seat.Failures(), 1, "the run still fails")
		})

		t.Run("fails the run at once for a damaged file", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "damaged.json"), "{")
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, failsFrom(900), prop.Seed(7), prop.Store(dir))
			want := fmt.Sprintf("prop: the store %s of %q: store: read damaged.json: store: not an entry",
				dir, contract)
			assert.HasPrefix(t, seat.Message(), want, "the message names the file")
			assert.Empty(t, seat.Failures(), "no run, so no record")
		})

		t.Run("fails the run at once for a store that cannot be read", func(t *testing.T) {
			t.Parallel()
			file := filepath.Join(t.TempDir(), "file")
			write(t, file, "")
			seat := newTestSeat(t.Name())
			dir := filepath.Join(file, "store")
			prop.ForAll(seat, contract, failsFrom(900), prop.Seed(7), prop.Store(dir))
			assert.HasPrefix(t, seat.Message(), fmt.Sprintf("prop: the store %s of %q: ", dir, contract),
				"the message names the store")
		})

		t.Run("fails a second property of a test with the same contract and store", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			seat := newTestSeat(t.Name())
			defer seat.end()
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Store(dir))
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Store(dir))
			assert.Equal(t, seat.Message(), duplicateMessage, "the second property is a problem of the test")
		})

		t.Run("releases a property's claim when its test ends", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			first := newTestSeat(t.Name())
			prop.ForAll(first, contract, draws(prop.Integer(0, 9)), prop.Store(dir))
			first.end()
			second := newTestSeat(t.Name())
			defer second.end()
			prop.ForAll(second, contract, draws(prop.Integer(0, 9)), prop.Store(dir))
			assert.False(t, second.Failed(), "the test of another run claims the contract again")
		})

		claims := []struct {
			name string
			seat func(string) assert.TB
			opts []prop.Option
		}{
			{
				name: "claims nothing for a run without a store",
				seat: func(name string) assert.TB { return newTestSeat(name) },
				opts: []prop.Option{prop.Store("")},
			},
			{
				name: "claims nothing for a seat without Cleanup",
				seat: func(string) assert.TB { return assert.NewRecorder() },
			},
		}
		for _, tt := range claims {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				seat := tt.seat(t.Name())
				opts := append([]prop.Option{prop.Store(t.TempDir())}, tt.opts...)
				prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), opts...)
				prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), opts...)
				failed, _ := seat.(interface{ Failed() bool })
				assert.False(t, failed.Failed(), "both properties run")
			})
		}

		t.Run("claims each contract of a store apart", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			seat := newTestSeat(t.Name())
			defer seat.end()
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Store(dir))
			prop.ForAll(seat, "another property holds", draws(prop.Integer(0, 9)), prop.Store(dir))
			assert.False(t, seat.Failed(), "both properties run")
		})
	})
}

// TestStoreEnv checks the default directory of a store, which the working
// directory decides. Each case changes the process's working directory, so
// the cases run one at a time.
func TestStoreEnv(t *testing.T) {
	t.Run("ForAll", func(t *testing.T) {
		t.Run("keeps the store of a seat with a name in testdata/prop under the name", func(t *testing.T) {
			t.Chdir(t.TempDir())
			seat := newTestSeat("TestRoundTrip/one case")
			defer seat.end()
			prop.ForAll(seat, contract, failsFrom(1001), prop.Seed(7))
			got := loaded(t, filepath.Join("testdata", "prop", "TestRoundTrip", "one case"))
			assert.Length(t, got.Entries, 1, "the entry of the minimal case")
		})

		t.Run("keeps no store for a seat without a name", func(t *testing.T) {
			t.Chdir(t.TempDir())
			prop.ForAll(assert.NewRecorder(), contract, failsFrom(1001), prop.Seed(7))
			assert.False(t, exists("testdata"), "no directory")
		})

		t.Run("keeps no store under Store of the empty string", func(t *testing.T) {
			t.Chdir(t.TempDir())
			seat := newTestSeat(t.Name())
			defer seat.end()
			prop.ForAll(seat, contract, failsFrom(1001), prop.Seed(7), prop.Store(""))
			assert.False(t, exists("testdata"), "no directory")
		})
	})
}

// failsFrom returns the body that draws an integer in [0, 10000], and ends
// the case at a value of least or more with a record whose identity an
// entry keeps: the assertion big and the contract fits.
func failsFrom(least int) func(*prop.Case) {
	g := prop.Integer(0, 10000)
	return func(c *prop.Case) {
		if c.Draw(g, drawn) >= least {
			c.Report(assert.Failure{Assertion: big, Contract: fits}, true)
		}
	}
}

// detached is the body that draws a digit and fails with a message from a
// goroutine without a frame of the caller's code, a failure that no entry
// can keep.
func detached(c *prop.Case) {
	c.Draw(prop.Integer(0, 9), drawn)
	go c.Errorf("detached")
	for len((*engine.Case)(c).Failures()) == 0 {
		runtime.Gosched()
	}
}

// entry returns the entry of the property contract for the case of one
// integer choice of value, which fails with the identity of [failsFrom],
// found earlier.
func entry(value uint64) store.Entry {
	literal, _ := store.Literal(int(value))
	return store.Entry{
		Definition:     "1.2.0",
		Property:       contract,
		Identity:       store.Identity{Assertion: big, Contract: fits},
		Choices:        []choice.Choice{integer(value)},
		Counterexample: []store.Draw{{Label: drawn, Value: literal}},
		Found:          earlier,
	}
}

// document returns the JSON of e as a file of a store states it.
func document(e store.Entry) string {
	data, _ := json.Marshal(e)
	return string(data)
}

// identities returns the identity of each entry.
func identities(entries []store.Entry) []store.Identity {
	out := make([]store.Identity, len(entries))
	for i, e := range entries {
		out[i] = e.Identity
	}
	return out
}

// valuesOf returns the label and the decoded JSON value of each draw, so
// that two spellings of one value compare equal.
func valuesOf(draws []store.Draw) [][2]any {
	out := make([][2]any, len(draws))
	for i, d := range draws {
		var v any
		_ = json.Unmarshal(d.Value, &v)
		out[i] = [2]any{d.Label, v}
	}
	return out
}

// save writes e to the store dir.
func save(t *testing.T, dir string, e store.Entry) {
	t.Helper()
	written, err := store.Save(dir, e)
	assert.NoError(t, err, "the entry is saved")
	assert.True(t, written, "the entry's file is new")
}

// loaded returns the entries of the property contract in the store dir.
func loaded(t *testing.T, dir string) store.Stored {
	t.Helper()
	stored, err := store.Load(dir, contract)
	assert.NoError(t, err, "the store reads")
	return stored
}

// write writes content to the file at path.
func write(t *testing.T, path, content string) {
	t.Helper()
	assert.NoError(t, os.WriteFile(path, []byte(content), fileMode), "the file is written")
}

// read returns the content of the file at path.
func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoError(t, err, "the file reads")
	return string(data)
}

// exists reports whether a file or a directory exists at path.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// detailOfOn runs body as the property contract on seat under opts, and
// returns the detail of the one record the run reported, or nil for a run
// that reported none.
func detailOfOn(seat *testSeat, body func(*prop.Case), opts ...prop.Option) map[string]any {
	prop.ForAll(seat, contract, body, opts...)
	records := seat.Failures()
	if len(records) == 0 {
		return nil
	}
	return records[0].Detail
}
