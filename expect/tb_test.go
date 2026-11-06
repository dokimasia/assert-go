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

// The compiler checks that the two seats of a test and the recorder satisfy
// TB.
var (
	_ expect.TB = (*testing.T)(nil)
	_ expect.TB = (*testing.B)(nil)
	_ expect.TB = (*expect.Recorder)(nil)
)

// tbContext keeps the context that a call of Context returns.
var tbContext any

// skippingSeat is a seat whose Skipf does nothing.
type skippingSeat struct{}

func (skippingSeat) Helper()               {}
func (skippingSeat) Fatalf(string, ...any) {}
func (skippingSeat) Errorf(string, ...any) {}

// Skipf drops the reason.
func (skippingSeat) Skipf(string, ...any) {}

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

	t.Run("Skipf", func(t *testing.T) {
		t.Parallel()

		t.Run("ends the call with a fault on a recorder, as the aborting surface does", func(t *testing.T) {
			t.Parallel()

			r := expect.NewRecorder()
			expect.Skipf(r, "the stack does not run")
			expect.Length(t, r.Faults(), 1, "one fault")
			expect.Equal(t, r.Message(), "assert.Skipf: the seat has no Skipf", "the fault of the aborting surface")
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

// TestTBAllocs checks the allocation ceilings of Context, Cleanup, Skipf and
// WriteRecord.
func TestTBAllocs(t *testing.T) {
	alloctest.Check(t, tbCases())
}

// BenchmarkTB measures Context, Cleanup, Skipf and WriteRecord.
func BenchmarkTB(b *testing.B) {
	for _, c := range tbCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// noCleanup is a function that the allocation case of Cleanup registers.
func noCleanup() {}

// tbCases returns the allocation cases of Context, Cleanup, Skipf and
// WriteRecord, each with its ceiling. Context reads the context of a recorder.
// Cleanup registers a function with a recorder, which then runs it. The
// recorder keeps the capacity of its list of cleanups, so the registration
// allocates nothing after the first call. Skipf runs on a seat whose Skipf
// does nothing. WriteRecord runs on a recorder, which writes nothing.
func tbCases() []alloctest.Case {
	seat := expect.NewRecorder().WithContext(context.Background())
	cleaning := expect.NewRecorder()
	return []alloctest.Case{
		{Name: "Context", Call: func(assert.TB) { tbContext = expect.Context(seat) }},
		{Name: "Cleanup", Call: func(assert.TB) {
			expect.Cleanup(cleaning, noCleanup)
			cleaning.RunCleanups()
		}},
		{Name: "Skipf", Call: func(assert.TB) { expect.Skipf(skippingSeat{}, "the stack does not run") }},
		{Name: "WriteRecord", Allocs: 2, Call: func(assert.TB) {
			expect.WriteRecord(seat, "dokimi.suite", `{"seed":"7"}`)
		}},
	}
}
