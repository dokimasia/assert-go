// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"context"
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
)

// Compile-time proof that both seats of a test and the recorder satisfy TB.
var (
	_ expect.TB = (*testing.T)(nil)
	_ expect.TB = (*testing.B)(nil)
	_ expect.TB = (*expect.Recorder)(nil)
)

// tbContext keeps the context that a call of Context returns.
var tbContext any

func TestTB(t *testing.T) {
	t.Parallel()

	t.Run("TB", func(t *testing.T) {
		t.Parallel()

		t.Run("is the seat of the aborting surface", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.TB](), reflect.TypeFor[assert.TB](), "one seat for both surfaces")
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the context of the seat, as the aborting surface does", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, expect.Context(t), t.Context(), "the context of the test")
		})
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Parallel()

		t.Run("registers f with the seat, as the aborting surface does", func(t *testing.T) {
			t.Parallel()

			r := expect.NewRecorder()
			ran := false
			expect.Cleanup(r, func() { ran = true })
			r.RunCleanups()
			expect.True(t, ran, "the recorder runs the function that Cleanup registered")
		})
	})
}

// TestTBAllocs checks the allocation ceilings of Context and Cleanup.
func TestTBAllocs(t *testing.T) {
	alloctest.Check(t, tbCases())
}

// BenchmarkTB measures Context and Cleanup.
func BenchmarkTB(b *testing.B) {
	for _, c := range tbCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// noCleanup is a function that the allocation case of Cleanup registers.
func noCleanup() {}

// tbCases returns a call of Context on a recorder that states a context, and a
// call of Cleanup on a recorder that runs the function, each with its
// allocation ceiling. The recorder keeps the capacity of its list of
// cleanups, so a registration that a run follows allocates nothing after the
// first call.
func tbCases() []alloctest.Case {
	seat := expect.NewRecorder().WithContext(context.Background())
	cleaning := expect.NewRecorder()
	return []alloctest.Case{
		{Name: "Context", Call: func(assert.TB) { tbContext = expect.Context(seat) }},
		{Name: "Cleanup", Call: func(assert.TB) {
			expect.Cleanup(cleaning, noCleanup)
			cleaning.RunCleanups()
		}},
	}
}
