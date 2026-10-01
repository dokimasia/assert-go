// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestIdentity checks what a failure is kept by, and the order in which
// failures of equal size shrink.
func TestIdentity(t *testing.T) {
	t.Parallel()

	t.Run("Compare", func(t *testing.T) {
		t.Parallel()

		at := func(file string, line int) assert.Where { return assert.Where{File: file, Line: line} }
		tests := []struct {
			name string
			a, b engine.Identity
			want int
		}{
			{
				name: "returns 0 for equal identities",
				a:    engine.Identity{Assertion: "equal", Where: at("a_test.go", 3)},
				b:    engine.Identity{Assertion: "equal", Where: at("a_test.go", 3)},
				want: 0,
			},
			{
				name: "returns -1 for an earlier assertion",
				a:    engine.Identity{Assertion: "equal", Where: at("b_test.go", 9)},
				b:    engine.Identity{Assertion: "true", Where: at("a_test.go", 3)},
				want: -1,
			},
			{
				name: "returns -1 for an earlier contract with one assertion",
				a:    engine.Identity{Assertion: "equal", Contract: "a total", Panic: "string"},
				b:    engine.Identity{Assertion: "equal", Contract: "b total", Panic: "*errors.errorString"},
				want: -1,
			},
			{
				name: "returns 1 for a later panic type with one assertion",
				a:    engine.Identity{Panic: "string", Where: at("a_test.go", 3)},
				b:    engine.Identity{Panic: "*errors.errorString", Where: at("b_test.go", 9)},
				want: 1,
			},
			{
				name: "returns -1 for an earlier file with one assertion and panic type",
				a:    engine.Identity{Assertion: "equal", Where: at("a_test.go", 9)},
				b:    engine.Identity{Assertion: "equal", Where: at("b_test.go", 3)},
				want: -1,
			},
			{
				name: "returns 1 for a later line in one file",
				a:    engine.Identity{Assertion: "equal", Where: at("a_test.go", 9)},
				b:    engine.Identity{Assertion: "equal", Where: at("a_test.go", 3)},
				want: 1,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.a.Compare(tt.b), tt.want, "the order of the identities")
			})
		}
	})

	t.Run("Replay", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the type of a panic's value at the line that raised it", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			e := engine.Replay(func(*engine.Case) { panic(errors.New(strconv.Itoa(site(&at)))) }, nil, nil)
			assert.Equal(t, e.Status, engine.CaseFailed, "a panic fails the case")
			assert.Equal(t, e.Identity, engine.Identity{Panic: "*errors.errorString", Where: at},
				"the panic's type and the caller's line")
		})

		t.Run("returns the frame of a standard library function that panics", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(*engine.Case) { _ = strings.Repeat("x", -1) }, nil, nil)
			assert.Equal(t, e.Identity.Panic, "string", "the type of the panic's message")
			inside := strings.HasSuffix(e.Identity.Where.File, "/strings/strings.go")
			assert.True(t, inside, "the frame inside strings.Repeat")
		})

		t.Run("returns the caller's line for a panic inside the module's root package", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			e := engine.Replay(func(*engine.Case) { assert.Equal(nil, site(&at), 0, "a seat that is nil") }, nil, nil)
			assert.Equal(t, e.Identity.Where, at, "the frames of the assertion are skipped")
		})

		t.Run("returns the caller's line for a panic inside a package of the module", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			e := engine.Replay(func(*engine.Case) { engine.Integer(site(&at), 0) }, nil, nil)
			assert.Equal(t, e.Identity.Where, at, "the frames of the engine are skipped")
		})

		third := assert.Where{File: "a_test.go", Line: 3}
		records := []struct {
			name string
			give assert.Failure
			want engine.Identity
		}{
			{
				name: "returns the assertion and the contract of a record without a location",
				give: assert.Failure{Assertion: "equal", Contract: "the totals match"},
				want: engine.Identity{Assertion: "equal", Contract: "the totals match"},
			},
			{
				name: "returns the assertion and the location of a record with a location",
				give: assert.Failure{Assertion: "equal", Contract: "the totals match", Where: third},
				want: engine.Identity{Assertion: "equal", Where: third},
			},
			{
				name: "returns no contract for a record without an assertion",
				give: assert.Failure{Contract: "a message"},
				want: engine.Identity{},
			},
		}
		for _, tt := range records {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e := engine.Replay(func(c *engine.Case) { c.Report(tt.give, false) }, nil, nil)
				assert.Equal(t, e.Identity, tt.want, "the identity of the record")
			})
		}
	})
}

// TestIdentityZeroAlloc checks that Compare allocates nothing.
func TestIdentityZeroAlloc(t *testing.T) {
	a, b := engine.Identity{Assertion: "equal"}, engine.Identity{Assertion: "true"}
	assert.MaxAllocs(t, func() { _ = a.Compare(b) }, 0, "Compare allocates nothing")
}

// BenchmarkIdentity measures Compare under a ceiling of no allocation.
func BenchmarkIdentity(b *testing.B) {
	first := engine.Identity{Assertion: "equal", Where: assert.Where{File: "a_test.go", Line: 3}}
	second := engine.Identity{Assertion: "equal", Where: assert.Where{File: "a_test.go", Line: 9}}

	b.Run("Compare", func(b *testing.B) {
		var got int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = first.Compare(second)
		}
		assert.Equal(b, got, -1, "the earlier line first")
	})
}
