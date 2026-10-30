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

	t.Run("WriteRecord", func(t *testing.T) {
		t.Parallel()

		t.Run("ends the call with a fault for a refused kind, as the aborting surface does", func(t *testing.T) {
			t.Parallel()

			r := expect.NewRecorder()
			expect.WriteRecord(r, "suite", `{}`)
			expect.Length(t, r.Faults(), 1, "one fault")
			expect.Equal(t, r.Message(), `assert.WriteRecord: kind: "suite" does not match dokimi\.[a-z][a-z0-9-]*`,
				"the fault of the aborting surface")
		})
	})
}

// TestTBAllocs checks the allocation ceilings of Context, Cleanup and
// WriteRecord.
func TestTBAllocs(t *testing.T) {
	alloctest.Check(t, tbCases())
}

// BenchmarkTB measures Context, Cleanup and WriteRecord.
func BenchmarkTB(b *testing.B) {
	for _, c := range tbCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// noCleanup is a function that the allocation case of Cleanup registers.
func noCleanup() {}

// tbCases returns a call of Context on a recorder that states a context, a
// call of Cleanup on a recorder that runs the function, and a call of
// WriteRecord on a recorder, which writes nothing, each with its allocation
// ceiling. The recorder keeps the capacity of its list of cleanups, so a
// registration that a run follows allocates nothing after the first call.
func tbCases() []alloctest.Case {
	seat := expect.NewRecorder().WithContext(context.Background())
	cleaning := expect.NewRecorder()
	return []alloctest.Case{
		{Name: "Context", Call: func(assert.TB) { tbContext = expect.Context(seat) }},
		{Name: "Cleanup", Call: func(assert.TB) {
			expect.Cleanup(cleaning, noCleanup)
			cleaning.RunCleanups()
		}},
		{Name: "WriteRecord", Allocs: 2, Call: func(assert.TB) {
			expect.WriteRecord(seat, "dokimi.suite", `{"seed":"7"}`)
		}},
	}
}
