// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/record"
)

// The compiler checks that the two seats of a test and the recorder satisfy
// TB.
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

// faultSeat is a seat that keeps each fault that it receives. It does not
// have Cleanup.
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

// skippingSeat is a seat whose Skipf does nothing.
type skippingSeat struct {
	bareSeat
}

// Skipf drops the reason.
func (skippingSeat) Skipf(string, ...any) {}

func TestTB(t *testing.T) {
	t.Parallel()

	t.Run("Helper", func(t *testing.T) {
		t.Parallel()

		t.Run("calls Helper of the seat through the interface", func(t *testing.T) {
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

		t.Run("calls Fatalf of the seat through the interface", func(t *testing.T) {
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

		t.Run("calls Errorf of the seat through the interface", func(t *testing.T) {
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

		t.Run("returns the context of a recorder", func(t *testing.T) {
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

	t.Run("Skipf", func(t *testing.T) {
		t.Parallel()

		t.Run("skips the test of a *testing.T with the reason at the line of the call", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				assert.Skipf(t, "the stack at %s does not run", "nats")
				t.Error("Skipf returned on a *testing.T")
				return
			}
			out, err := childtest.Run(t, t.Name())
			if err != nil || !strings.Contains(out, "--- SKIP: "+t.Name()+" ") {
				t.Fatalf("the child exits with %v, want a skip:\n%s", err, out)
			}
			if !regexp.MustCompile(`tb_test\.go:\d+: the stack at nats does not run`).MatchString(out) {
				t.Fatalf("the child logged no reason at the line of the call:\n%s", out)
			}
		})

		t.Run("ends the check of Rejects on a wrapper, and Rejects ends its call as skipped", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			failures := assert.Rejects(r, "the check needs the stack", func(tb assert.TB) {
				assert.Skipf(wrapped{tb}, "the stack does not run")
			})
			got := decoded(t, r.Records())
			if len(failures) != 0 || len(got) != 1 || got[0]["assertion"] != "rejects" ||
				got[0]["verdict"] != "skip" || got[0]["reason"] != "the stack does not run" || r.Failed() {

				t.Fatalf("Rejects returned %v and recorded %v, want no record and the skip of rejects",
					failures, got)
			}
		})

		t.Run("ends the call with a fault on a recorder", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Skipf(r, "the stack does not run")
			faults := r.Faults()
			var got *fault.Error
			if len(faults) != 1 || !errors.As(faults[0], &got) || got.Op != "assert.Skipf" ||
				got.Reason != "the seat has no Skipf" || !r.Failed() {

				t.Fatalf("the recorder kept the faults %v and failed: %v, want the fault of assert.Skipf",
					faults, r.Failed())
			}
		})
	})

	t.Run("WriteRecord", func(t *testing.T) {
		t.Parallel()

		t.Run("writes each record of a kind into the output of the test, numbered from 1", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				assert.WriteRecord(t, "dokimi.suite", `{"seed":"7"}`)
				assert.WriteRecord(wrapped{t}, "dokimi.suite", `{"seed":"8"}`)
				assert.WriteRecord(t, "dokimi.suite", "{\n\"seed\":\"9\"}")
				return
			}
			out, err := childtest.Run(t, t.Name())
			if err != nil || !strings.Contains(out, "--- PASS: "+t.Name()+" ") {
				t.Fatalf("the child exits with %v, want a pass:\n%s", err, out)
			}
			for _, want := range []string{
				"=== ATTR  " + t.Name() + ` dokimi.suite.1 {"seed":"7"}`,
				"=== ATTR  " + t.Name() + ` dokimi.suite.2 {"seed":"8"}`,
				"=== ATTR  " + t.Name() + ` dokimi.suite.3 { "seed":"9"}`,
			} {
				if !strings.Contains(out, want) {
					t.Fatalf("the child wrote no %q, without DOKIMI_ASSERT_RECORD:\n%s", want, out)
				}
			}
		})

		t.Run("writes nothing on a recorder", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.WriteRecord(r, "dokimi.suite", `{}`)
			if r.Failed() || len(r.Records()) != 0 {
				t.Fatalf("the recorder failed: %v, and kept %v, want neither", r.Failed(), r.Records())
			}
		})

		t.Run("ends the call with a fault of WriteRecord for the kind of the call records", func(t *testing.T) {
			t.Parallel()

			seat := &faultSeat{}
			assert.WriteRecord(seat, "dokimi.assert", `{}`)

			var got *fault.Error
			if len(seat.faults) != 1 || !errors.As(seat.faults[0], &got) || got.Op != "assert.WriteRecord" ||
				got.Reason != "dokimi.assert is the kind of the call records of assert" {

				t.Fatalf("WriteRecord reported %v, want the fault of assert.WriteRecord", seat.faults)
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

// tbCases returns the allocation cases of Context, Cleanup, Skipf and
// WriteRecord, each with its ceiling. Context reads the context of a recorder.
// Cleanup and Skipf run on seats whose methods do nothing. WriteRecord runs on
// a recorder, which writes nothing.
func tbCases() []alloctest.Case {
	seat := assert.NewRecorder().WithContext(context.Background())
	return []alloctest.Case{
		{Name: "Context", Call: func(assert.TB) { tbContext = assert.Context(seat) }},
		{Name: "Cleanup", Call: func(assert.TB) { assert.Cleanup(cleaningSeat{}, noCleanup) }},
		{Name: "Skipf", Call: func(assert.TB) { assert.Skipf(skippingSeat{}, "the stack does not run") }},
		{Name: "WriteRecord", Allocs: 2, Call: func(assert.TB) {
			assert.WriteRecord(seat, "dokimi.suite", `{"seed":"7"}`)
		}},
	}
}
