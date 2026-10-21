// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package wrap_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/wrap"
)

// wrapper is a seat that wraps another seat, as the handle of a check does.
// It has the three methods of the wrapped seat and Unwrap.
type wrapper struct {
	assert.TB
}

// Unwrap returns the wrapped seat.
func (w wrapper) Unwrap() assert.TB { return w.TB }

// labelled is a seat that wraps another seat and has a label.
type labelled struct {
	wrapper

	label string
}

// Label returns the seat's label.
func (s labelled) Label() string { return s.label }

// label is the method that the cases search a chain for.
type label interface {
	// Label returns the seat's label.
	Label() string
}

// found keeps what a case of the allocation table found, so the compiler
// keeps the call.
var found any

func TestWrap(t *testing.T) {
	t.Parallel()

	t.Run("As", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the seat itself when it has the method", func(t *testing.T) {
			t.Parallel()
			got, ok := wrap.As[label](labelled{label: "outer"})
			assert.True(t, ok, "the seat has the method")
			assert.Equal(t, got.Label(), "outer", "the seat itself")
		})
		t.Run("returns the first seat of the chain that has the method", func(t *testing.T) {
			t.Parallel()
			seat := wrapper{labelled{wrapper: wrapper{labelled{label: "inner"}}, label: "middle"}}
			got, ok := wrap.As[label](seat)
			assert.True(t, ok, "a seat of the chain has the method")
			assert.Equal(t, got.Label(), "middle", "the seat nearest to the start of the chain")
		})
		t.Run("reports false for a chain without the method", func(t *testing.T) {
			t.Parallel()
			_, ok := wrap.As[label](wrapper{wrapper{assert.NewRecorder()}})
			assert.False(t, ok, "no seat of the chain has the method")
		})
		t.Run("stops at a seat whose Unwrap returns nil", func(t *testing.T) {
			t.Parallel()
			_, ok := wrap.As[*assert.Recorder](wrapper{})
			assert.False(t, ok, "the chain ends at the wrapper")
		})
		t.Run("reports false for a nil seat", func(t *testing.T) {
			t.Parallel()
			_, ok := wrap.As[label](nil)
			assert.False(t, ok, "a nil seat has no method")
		})
	})

	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the seat that the method Unwrap of a seat returns", func(t *testing.T) {
			t.Parallel()
			r := assert.NewRecorder()
			assert.True(t, wrap.Unwrap(wrapper{r}) == any(r), "the wrapped recorder")
		})
		t.Run("returns nil for a seat without Unwrap", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, wrap.Unwrap(assert.NewRecorder()), "no wrapped seat")
		})
		t.Run("returns nil for a seat whose Unwrap returns nil", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, wrap.Unwrap(wrapper{}), "no wrapped seat")
		})
	})

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("panics for a nil function", func(t *testing.T) {
			t.Parallel()
			raised := assert.Panics(t, func() { wrap.Register(nil) }, "a nil function registers nothing")
			assert.Equal(t, raised, any("wrap: Register registers no function"), "the message names the call")
		})
		t.Run("panics for a second registration and keeps the first", func(t *testing.T) {
			t.Parallel()
			raised := assert.Panics(t, func() { wrap.Register(func(any) any { return nil }) },
				"the root package registered its function while it initialised")
			assert.Equal(t, raised, any("wrap: Register registers a second function"), "the message names the call")
			r := assert.NewRecorder()
			assert.True(t, wrap.Unwrap(wrapper{r}) == any(r), "the function of the root package stays registered")
		})
	})
}

// TestWrapAllocs checks the allocation ceilings of As and Unwrap.
func TestWrapAllocs(t *testing.T) {
	alloctest.Check(t, wrapCases())
}

// BenchmarkWrap measures As and Unwrap.
func BenchmarkWrap(b *testing.B) {
	for _, c := range wrapCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// wrapCases returns a call of As and of Unwrap on a chain of two seats, with
// its allocation ceiling. The chain is an interface value already, so a call
// converts nothing.
func wrapCases() []alloctest.Case {
	var seat any = wrapper{labelled{label: "inner"}}
	return []alloctest.Case{
		{Name: "As", Call: func(assert.TB) { found, _ = wrap.As[label](seat) }},
		{Name: "Unwrap", Call: func(assert.TB) { found = wrap.Unwrap(seat) }},
	}
}
