// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/record"
)

// Compile-time proof that both seats satisfy TB. A runtime nil check
// cannot state this: the compiler already knows the value is non-nil.
var (
	_ assert.TB = (*testing.T)(nil)
	_ assert.TB = (*testing.B)(nil)
	_ assert.TB = (*assert.Recorder)(nil)
)

// tbContext keeps the context that a call of Context returns.
var tbContext any

// bareSeat is a seat with the three methods of TB and nothing else, as the
// seat of a generated check body can be.
type bareSeat struct{}

func (bareSeat) Helper()               {}
func (bareSeat) Fatalf(string, ...any) {}
func (bareSeat) Errorf(string, ...any) {}

// wrapped is a seat that wraps another seat, as the handle of a check does.
// It has the three methods of the wrapped seat and Unwrap.
type wrapped struct {
	assert.TB
}

// Unwrap returns the wrapped seat.
func (w wrapped) Unwrap() assert.TB { return w.TB }

// faultSeat is a seat that keeps each fault that it receives, and has no
// Cleanup.
type faultSeat struct {
	bareSeat

	faults []error
}

// ReportFault keeps err.
func (s *faultSeat) ReportFault(err error, _ bool) { s.faults = append(s.faults, err) }

// cleaningSeat is a seat whose Cleanup does nothing.
type cleaningSeat struct {
	bareSeat
}

// Cleanup drops f.
func (cleaningSeat) Cleanup(func()) {}

func TestTB(t *testing.T) {
	t.Parallel()

	t.Run("Helper", func(t *testing.T) {
		t.Parallel()

		t.Run("reaches the seat through the interface", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			var seat assert.TB = r
			seat.Helper()

			if got, want := r.HelperCalls(), 1; got != want {
				t.Fatalf("HelperCalls() = %d, want %d", got, want)
			}
		})
	})

	t.Run("Fatalf", func(t *testing.T) {
		t.Parallel()

		t.Run("reaches the seat through the interface", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			var seat assert.TB = r
			seat.Fatalf("through the interface")

			if got, want := r.Message(), "through the interface"; got != want {
				t.Fatalf("Message() = %q, want %q", got, want)
			}
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		t.Parallel()

		t.Run("reaches the seat through the interface", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			var seat assert.TB = r
			seat.Errorf("through the interface")

			if got, want := len(r.Messages()), 1; got != want {
				t.Fatalf("len(Errors()) = %d, want %d", got, want)
			}
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the context of a *testing.T", func(t *testing.T) {
			t.Parallel()

			if got := assert.Context(t); got != t.Context() {
				t.Fatalf("Context(t) = %v, want t.Context()", got)
			}
		})

		t.Run("returns the context that a recorder states", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(t.Context(), ledgerKey{}, "ledger")
			if got := assert.Context(assert.NewRecorder().WithContext(ctx)); got != ctx {
				t.Fatalf("Context of the recorder = %v, want the context that WithContext set", got)
			}
		})

		t.Run("returns context.Background() for a seat without a context", func(t *testing.T) {
			t.Parallel()

			want := context.Background() //nolint:usetesting // Context returns it for a seat without a context
			if got := assert.Context(bareSeat{}); got != want {
				t.Fatalf("Context of a bare seat = %v, want context.Background()", got)
			}
		})

		t.Run("returns the context of the seat that a seat wraps", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(t.Context(), ledgerKey{}, "ledger")
			if got := assert.Context(wrapped{assert.NewRecorder().WithContext(ctx)}); got != ctx {
				t.Fatalf("Context of the wrapper = %v, want the context of the recorder that it wraps", got)
			}
		})

		t.Run("returns context.Background() for a seat whose Unwrap returns nil", func(t *testing.T) {
			t.Parallel()

			want := context.Background() //nolint:usetesting // Context returns it for a chain without a context
			if got := assert.Context(wrapped{}); got != want {
				t.Fatalf("Context of a wrapper of no seat = %v, want context.Background()", got)
			}
		})
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Parallel()

		t.Run("registers f with the test of a *testing.T", func(t *testing.T) {
			t.Parallel()

			ran := false
			t.Cleanup(func() {
				if !ran {
					t.Error("the test ended without the function that Cleanup registered")
				}
			})
			assert.Cleanup(t, func() { ran = true })
		})

		t.Run("registers f with a recorder, which runs it at RunCleanups", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			ran := false
			assert.Cleanup(r, func() { ran = true })
			r.RunCleanups()

			if !ran {
				t.Fatal("RunCleanups did not run the function that Cleanup registered")
			}
		})

		t.Run("registers f with the first seat of the chain that has the method", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			ran := false
			assert.Cleanup(wrapped{wrapped{r}}, func() { ran = true })
			r.RunCleanups()

			if !ran {
				t.Fatal("the wrapped recorder did not run the function that Cleanup registered")
			}
		})

		t.Run("ends the call with a fault on a chain without the method", func(t *testing.T) {
			t.Parallel()

			seat := &faultSeat{}
			ran := false
			assert.Cleanup(wrapped{seat}, func() { ran = true })

			var got *fault.Error
			if len(seat.faults) != 1 || !errors.As(seat.faults[0], &got) || got.Op != "assert.Cleanup" ||
				got.Reason != "the seat has no Cleanup" || ran {

				t.Fatalf("Cleanup reported %v and ran f: %v, want the fault of assert.Cleanup and no run",
					seat.faults, ran)
			}
		})
	})

	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps the call record of an assertion on a wrapper in the wrapped recorder", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Equal(wrapped{wrapped{r}}, 1, 1, "one is one")
			if got := decoded(t, r.Records()); len(got) != 1 || got[0]["assertion"] != "equal" {
				t.Fatalf("Records() = %v, want the record of the call of equal", got)
			}
		})

		t.Run("reports a failure on a wrapper as a record to the recorder that it wraps", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Equal(wrapped{r}, 1, 2, "one is two")
			if got := r.Failures(); len(got) != 1 || got[0].Assertion != "equal" {
				t.Fatalf("Failures() = %v, want the record of the failure of equal", got)
			}
		})

		t.Run("records a check of Rejects on a wrapper of its seat under the call of Rejects", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			failures := assert.Rejects(r, "the check fails", func(tb assert.TB) {
				assert.Equal(wrapped{tb}, 1, 2, "one is two")
			})
			got := decoded(t, r.Records())
			if len(got) != 2 || got[0]["assertion"] != "rejects" || got[1]["assertion"] != "equal" ||
				got[1]["parent"] != 1.0 {

				t.Fatalf("Records() = %v, want the call of rejects and the check's call of equal under it", got)
			}
			if len(failures) != 1 || failures[0].Assertion != "equal" {
				t.Fatalf("Rejects returned %v, want the record of the check's failure", failures)
			}
		})

		t.Run("writes the call record of an assertion on a wrapper of a test's seat into the test's output",
			func(t *testing.T) {
				t.Parallel()
				if childtest.InChild(t) {
					assert.True(wrapped{t}, true, "the call on the wrapper")
					return
				}
				out, err := childtest.Run(t, t.Name(), record.Variable+"=1")
				if err != nil || !strings.Contains(out, "--- PASS: "+t.Name()+" ") {
					t.Fatalf("the child exits with %v, want a pass:\n%s", err, out)
				}
				if !strings.Contains(out, "=== ATTR  "+t.Name()+" dokimi.assert.1 ") {
					t.Fatalf("the child wrote no record of the call on the wrapper:\n%s", out)
				}
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

// tbCases returns a call of Context on a recorder that states a context, and a
// call of Cleanup on a seat whose Cleanup does nothing, each with its
// allocation ceiling.
func tbCases() []alloctest.Case {
	seat := assert.NewRecorder().WithContext(context.Background())
	return []alloctest.Case{
		{Name: "Context", Call: func(assert.TB) { tbContext = assert.Context(seat) }},
		{Name: "Cleanup", Call: func(assert.TB) { assert.Cleanup(cleaningSeat{}, noCleanup) }},
	}
}
