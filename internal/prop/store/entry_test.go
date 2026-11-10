// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"encoding/json"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/store"
)

// The allocation ceilings of an entry's methods.
const (
	// nameAllocs is the ceiling of the allocations of Name: the token, the
	// text that joins it to the contract, and the name.
	nameAllocs = 4
	// marshalAllocs is the ceiling of the allocations of MarshalJSON on the
	// pinned entry.
	marshalAllocs = 8
)

// TestEntry checks an entry's identity, its name and its JSON against what
// the definition's executable reference writes.
func TestEntry(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give store.Identity
			want bool
		}{
			{
				name: "reports true for an assertion's record with a location",
				give: store.Identity{Assertion: "equal", File: "a_test.go", Line: 3},
				want: true,
			},
			{
				name: "reports true for an assertion's record without a location",
				give: store.Identity{Assertion: "equal", Contract: "the totals match"},
				want: true,
			},
			{name: "reports true for a message", give: store.Identity{File: "a_test.go", Line: 3}, want: true},
			{
				name: "reports true for a raised error",
				give: store.Identity{Error: "*errors.errorString", File: "a_test.go", Line: 3},
				want: true,
			},
			{
				name: "reports true for the largest line",
				give: store.Identity{File: "a_test.go", Line: 2147483647},
				want: true,
			},
			{
				name: "reports false for a line past 2^31 - 1",
				give: store.Identity{File: "a_test.go", Line: 2147483648},
				want: false,
			},
			{name: "reports false for line 0", give: store.Identity{File: "a_test.go"}, want: false},
			{name: "reports false for no file", give: store.Identity{Line: 3}, want: false},
			{
				name: "reports false for a file with a slash",
				give: store.Identity{File: "pkg/a_test.go", Line: 3},
				want: false,
			},
			{
				name: "reports false for a file with a backslash",
				give: store.Identity{File: `pkg\a_test.go`, Line: 3},
				want: false,
			},
			{
				name: "reports false for an assertion and an error",
				give: store.Identity{Assertion: "equal", Error: "string", File: "a_test.go", Line: 3},
				want: false,
			},
			{
				name: "reports false for a contract without an assertion",
				give: store.Identity{Contract: "the totals match"},
				want: false,
			},
			{
				name: "reports false for a contract with an error",
				give: store.Identity{Assertion: "equal", Contract: "the totals match", Error: "string"},
				want: false,
			},
			{
				name: "reports false for a contract with a file",
				give: store.Identity{Assertion: "equal", Contract: "the totals match", File: "a_test.go"},
				want: false,
			},
			{
				name: "reports false for a contract with a line",
				give: store.Identity{Assertion: "equal", Contract: "the totals match", Line: 3},
				want: false,
			},
			{name: "reports false for no field", give: store.Identity{}, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the identity has a shape")
			})
		}
	})

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the name that the definition pins", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, pinned().Name(), pinnedName, "the mix of the contract and the token")
		})

		t.Run("returns another name for another contract", func(t *testing.T) {
			t.Parallel()
			other := pinned()
			other.Property = "encoding is deterministic"
			assert.Equal(t, other.Name(), "110988f43e88e891.json", "the second store vector's name")
		})

		t.Run("returns the name that the definition pins for no choices", func(t *testing.T) {
			t.Parallel()
			empty := store.Entry{Property: "an empty request is refused"}
			assert.Equal(t, empty.Name(), "3a15add8c7f746d1.json", "the third store vector's name")
		})

		t.Run("returns the name that the definition pins for an entry with steps", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, stepped().Name(), steppedName, "the name of the vector of format 2")
		})
	})

	t.Run("Entries", func(t *testing.T) {
		t.Parallel()

		all := []string{"capacity", "put", "v", "put", "v", "get"}
		tests := []struct {
			name      string
			giveBreak int
			want      []string
		}{
			{name: "returns each step before the draws that the case drew after it", giveBreak: -1, want: all},
			{name: "stops at a draw where the loop breaks", giveBreak: 0, want: all[:1]},
			{name: "stops at a step before a draw where the loop breaks", giveBreak: 1, want: all[:2]},
			{name: "stops at a step after the last draw where the loop breaks", giveBreak: 5, want: all},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got []string
				for d, s := range stepped().Entries() {
					if s != nil {
						got = append(got, s.Action)
					} else {
						got = append(got, d.Label)
					}
					if len(got)-1 == tt.giveBreak {
						break
					}
				}
				assert.Equal(t, got, tt.want, "the labels of the draws and the actions of the steps")
			})
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the object that the definition pins in its order", func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(pinned())
			assert.NoError(t, err, "the entry is JSON")
			assert.Equal(t, string(got), pinnedJSON, "the pinned entry")
		})

		t.Run("returns an empty list for no draws", func(t *testing.T) {
			t.Parallel()
			e := pinned()
			e.Counterexample = nil
			got, err := json.Marshal(e)
			assert.NoError(t, err, "the entry is JSON")
			assert.Contains(t, string(got), `"counterexample":[]`, "an empty list and not null")
		})

		t.Run("returns a draw without a value by its label alone", func(t *testing.T) {
			t.Parallel()
			e := pinned()
			e.Counterexample = []store.Draw{{Label: "conn"}}
			got, err := json.Marshal(e)
			assert.NoError(t, err, "the entry is JSON")
			assert.Contains(t, string(got), `"counterexample":[{"label":"conn"}]`, "the label alone")
		})

		t.Run("returns the UTC date of the time found", func(t *testing.T) {
			t.Parallel()
			e := pinned()
			e.Found = time.Date(2026, time.October, 1, 23, 30, 0, 0, time.FixedZone("UTC-5", -5*60*60))
			got, err := json.Marshal(e)
			assert.NoError(t, err, "the entry is JSON")
			assert.Contains(t, string(got), `"found":"2026-10-02"`, "half past four in the morning, UTC")
		})

		t.Run("returns the object of format 2 that the definition pins for an entry with steps", func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(stepped())
			assert.NoError(t, err, "the entry is JSON")
			assert.Equal(t, string(got), steppedJSON, "the steps among the draws")
		})

		t.Run("returns a step that the case took before its first draw first", func(t *testing.T) {
			t.Parallel()
			e := store.Entry{
				Counterexample: []store.Draw{{Label: "v"}},
				Steps:          []store.Step{{Action: "open", Client: -1}},
			}
			got, err := json.Marshal(e)
			assert.NoError(t, err, "the entry is JSON")
			assert.HasPrefix(t, string(got), `{"store":2,`, "an entry of format 2")
			assert.Contains(t, string(got), `"counterexample":[{"step":"open"},{"label":"v"}]`, "the step first")
		})

		t.Run("returns the steps of an entry without draws in order", func(t *testing.T) {
			t.Parallel()
			e := store.Entry{Steps: []store.Step{{Action: "open", Client: 0}, {Action: "close", Client: -1}}}
			got, err := json.Marshal(e)
			assert.NoError(t, err, "the entry is JSON")
			assert.Contains(t, string(got), `"counterexample":[{"step":"open","client":0},{"step":"close"}]`,
				"the client 0 and a sequential step")
		})

		t.Run("returns an error for a value that is not JSON", func(t *testing.T) {
			t.Parallel()
			e := pinned()
			e.Counterexample = []store.Draw{{Label: "broken", Value: json.RawMessage("{")}}
			_, err := json.Marshal(e)
			assert.HasError(t, err, "the value cannot be written")
		})
	})
}

// TestEntryAllocs checks that Valid and Entries allocate nothing, and the
// ceilings of Name and MarshalJSON.
func TestEntryAllocs(t *testing.T) {
	id := store.Identity{Assertion: "equal", File: "a_test.go", Line: 3}
	e, s := pinned(), stepped()
	assert.MaxAllocs(t, func() { _ = id.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() {
		for range s.Entries() {
		}
	}, 0, "Entries allocates nothing")
	assert.MaxAllocs(t, func() { _ = e.Name() }, nameAllocs, "Name allocates the token and the name")
	assert.MaxAllocs(t, func() { _, _ = e.MarshalJSON() }, marshalAllocs, "MarshalJSON allocates the JSON")
}

// BenchmarkEntry measures each method of an entry and of its identity, and
// Entries on the entry of format 2.
func BenchmarkEntry(b *testing.B) {
	e := pinned()

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = e.Identity.Valid()
		}
		assert.True(b, got, "the pinned identity has a shape")
	})

	b.Run("Entries", func(b *testing.B) {
		var got int
		s := stepped()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = 0
			for range s.Entries() {
				got++
			}
		}
		assert.Equal(b, got, 6, "the three draws and the three steps")
	})

	b.Run("Name", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(nameAllocs)
		defer c.End()
		for c.Loop() {
			got = e.Name()
		}
		assert.Equal(b, got, pinnedName, "the pinned name")
	})

	b.Run("MarshalJSON", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(marshalAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = e.MarshalJSON()
		}
		assert.Equal(b, string(got), pinnedJSON, "the pinned entry")
	})
}
