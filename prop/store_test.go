// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/prop"
)

// decodedReason is the reason of the fault on a stored case that decodes
// to other values than it records.
const decodedReason = "the stored case decodes to other values than it records, and the run tested those"

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
			assert.Equal(t, got.Name(), want.Name(), "the name of the contract and the minimal case's token")
			assert.Equal(t, got.Definition, conformance.Version(), "the version of the vendored definition")
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

		t.Run(
			"writes a value that Of derives as the typed literal of its shape, which Draws runs back",
			func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				body := func(c *prop.Case) {
					if c.Draw(prop.Of[order](), drawn).ID >= 5 {
						c.Report(assert.Failure{Assertion: big, Contract: fits}, true)
					}
				}
				seat := newTestSeat(t.Name())
				prop.ForAll(seat, contract, body, prop.Seed(7), prop.Store(dir))
				assert.Empty(t, seat.Faults(), "the store keeps the entry")
				entries := loaded(t, dir).Entries
				assert.Length(t, entries, 1, "one entry")
				stated := entries[0].Counterexample[0].Value
				assert.Equal(t, jsonTree(t, string(stated)), jsonTree(t, `{"type": "record", "fields": [`+
					`["id", {"type": "int", "value": 5}], ["lines", {"type": "list", "items": []}], ["note", {"type": "null"}]]}`),
					"the record of the minimal order")
				again := detailOf(body, prop.Seed(7), prop.Explain(false),
					prop.Draws(`[{"label": "value", "value": `+string(stated)+`}]`))
				assert.Equal(t, again[casesField], any(0), "the case of the entry fails first")
				assert.Equal(t, again[counterexampleField], any([]prop.Drawn{{Label: drawn, Value: order{ID: 5}}}),
					"the order that the entry states")
			},
		)

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

		t.Run("notes the fault of a file of a later format, and passes the run", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Seed(7), prop.Store(dir))
			assert.False(t, seat.Failed(), "the run passes")
			expectOnlyFault(t, seat.Faults(), laterFault(forAllOp, dir, "later.json", "2"))
		})

		t.Run("notes the fault of a stored case that decodes to other values than it records", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			moved := entry(950)
			moved.Counterexample[0].Value = json.RawMessage(`{"type": "int", "value": 951}`)
			save(t, dir, moved)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, failsFrom(900), prop.Seed(7), prop.Store(dir))
			expectOnlyFault(t, seat.Faults(), decodedFault(dir, moved))
		})

		tests := []struct {
			name    string
			changed func(*store.Entry)
		}{
			{
				name: "notes no fault of a stored value that another spelling of the same JSON states",
				changed: func(e *store.Entry) {
					e.Counterexample[0].Value = json.RawMessage(`{"value":950,"type":"int"}`)
				},
			},
			{
				name:    "notes no fault of a stored draw without a value",
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
				assert.Empty(t, seat.Faults(), "no fault")
			})
		}

		differing := []struct {
			name    string
			changed func(*store.Entry)
		}{
			{name: "notes the fault of a stored draw of another label", changed: func(e *store.Entry) {
				e.Counterexample[0].Label = "other"
			}},
			{name: "notes the fault of another number of stored draws", changed: func(e *store.Entry) {
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
				expectOnlyFault(t, seat.Faults(), decodedFault(dir, e))
			})
		}

		t.Run("notes the fault of a replayed draw without a typed literal, and passes the run", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			e := entry(950)
			save(t, dir, e)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, draws(prop.Just(complex(1, 2))), prop.Seed(7), prop.Store(dir))
			expectOnlyFault(t, seat.Faults(), decodedFault(dir, e))
			assert.False(t, seat.Failed(), "the run passes")
		})

		t.Run("notes the fault of a store that cannot keep the case, and fails the run", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, detached, prop.Seed(7), prop.Store(dir))
			faults := seat.Faults()
			expectOnlyFault(t, faults, fault.Error{
				Op:     forAllOp,
				Path:   fault.Path{fault.Field(dir)},
				Reason: fmt.Sprintf("the store keeps no case of %q", contract),
			})
			assert.ErrorIs(t, faults[0], store.ErrInvalid, "the store refuses an entry that no reader replays")
			assert.Length(t, seat.Records(), 1, "the run still fails")
		})

		t.Run("fails the run at once for a damaged file", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "damaged.json"), "{")
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, failsFrom(900), prop.Seed(7), prop.Store(dir))
			faults := seat.Faults()
			expectOnlyFault(t, faults, fault.Error{Op: forAllOp, Path: fault.Path{fault.Field(dir)}})
			expectFault(t, errors.Unwrap(faults[0]), fault.Error{
				Path:   fault.Path{fault.Field("damaged.json")},
				Kind:   store.ErrDamaged,
				Reason: "the file is not one JSON object",
			})
			assert.Empty(t, seat.Records(), "no run, so no record")
		})

		t.Run("fails the run at once for a store that cannot be read", func(t *testing.T) {
			t.Parallel()
			file := filepath.Join(t.TempDir(), "file")
			write(t, file, "")
			seat := newTestSeat(t.Name())
			dir := filepath.Join(file, "store")
			prop.ForAll(seat, contract, failsFrom(900), prop.Seed(7), prop.Store(dir))
			expectOnlyFault(t, seat.Faults(), fault.Error{
				Op:     forAllOp,
				Path:   fault.Path{fault.Field(dir)},
				Reason: "the store cannot be listed",
			})
		})

		t.Run("fails a second property of a test with the same contract and store", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			seat := newTestSeat(t.Name())
			defer seat.end()
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Store(dir))
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Store(dir))
			expectOnlyFault(t, seat.Faults(), fault.Error{
				Op:     forAllOp,
				Path:   fault.Path{fault.Field(dir)},
				Reason: duplicateReason,
			})
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

// entry returns the entry of the property contract for the case of one
// integer choice of value, which fails with the identity of [failsFrom],
// found earlier.
func entry(value uint64) store.Entry {
	lit, _ := literal.Encode(int(value))
	return store.Entry{
		Definition:     "1.2.0",
		Property:       contract,
		Identity:       store.Identity{Assertion: big, Contract: fits},
		Choices:        []choice.Choice{integer(value)},
		Counterexample: []store.Draw{{Label: drawn, Value: lit}},
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
	records := seat.Records()
	if len(records) == 0 {
		return nil
	}
	return records[0].Detail
}

// decodedFault returns the fault of ForAll at the file of e in the store
// dir, a stored case that decodes to other values than e records.
func decodedFault(dir string, e store.Entry) fault.Error {
	return fault.Error{Op: forAllOp, Path: fault.Path{fault.Field(dir), fault.Field(e.Name())}, Reason: decodedReason}
}
