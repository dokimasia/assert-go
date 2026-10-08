// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/record"
)

// The results of the allocation cases, which keep each call.
var (
	recorder        *assert.Recorder
	failures        []assert.Failure
	lines           []string
	messages        []string
	message         string
	failed          bool
	helpers         int
	recorderClock   assert.Clock
	recorderContext any
)

// TestRecorder checks what a recorder records of each call of its seat.
func TestRecorder(t *testing.T) {
	t.Parallel()

	t.Run("Fatalf", func(t *testing.T) {
		t.Parallel()

		t.Run("records the message instead of aborting", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("boom %d", 7)

			if !r.Failed() {
				t.Fatal("Failed() = false, want true")
			}
			if got, want := r.Message(), "boom 7"; got != want {
				t.Fatalf("Message() = %q, want %q", got, want)
			}
		})

		t.Run("keeps the first message", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("first")
			r.Fatalf("second")

			if got, want := r.Message(), "first"; got != want {
				t.Fatalf("Message() = %q, want %q", got, want)
			}
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		t.Parallel()

		t.Run("records every message", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Errorf("one")
			r.Errorf("two")

			if got, want := len(r.Messages()), 2; got != want {
				t.Fatalf("len(Messages()) = %d, want %d", got, want)
			}
		})

		t.Run("marks the recorder failed", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Errorf("one")

			if !r.Failed() {
				t.Fatal("Failed() = false, want true")
			}
		})
	})

	t.Run("Failed", func(t *testing.T) {
		t.Parallel()

		t.Run("returns false before anything is recorded", func(t *testing.T) {
			t.Parallel()

			if assert.NewRecorder().Failed() {
				t.Fatal("Failed() = true on a fresh recorder")
			}
		})
	})

	t.Run("HelperCalls", func(t *testing.T) {
		t.Parallel()

		t.Run("counts each call", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Helper()
			r.Helper()

			if got, want := r.HelperCalls(), 2; got != want {
				t.Fatalf("HelperCalls() = %d, want %d", got, want)
			}
		})
	})

	t.Run("Records", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nothing for a recorder that received no call", func(t *testing.T) {
			t.Parallel()

			if got := assert.NewRecorder().Records(); len(got) != 0 {
				t.Fatalf("Records() = %q, want none", got)
			}
		})

		t.Run("returns the call record of every call in the order of their numbers", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.True(r, true, "the first call")
			assert.Equal(r, 1, 2, "the second call")

			got := decoded(t, r.Records())
			if len(got) != 2 {
				t.Fatalf("Records() returns %d records, want 2", len(got))
			}
			if got[0]["seq"] != 1.0 || got[0]["assertion"] != "true" || got[0]["verdict"] != "pass" {
				t.Fatalf("the first record is %v, want a pass of true as 1", got[0])
			}
			if got[1]["seq"] != 2.0 || got[1]["assertion"] != "equal" || got[1]["verdict"] != "fail" {
				t.Fatalf("the second record is %v, want a failure of equal as 2", got[1])
			}
		})

		t.Run("returns the calls of a body under the call that ran it", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Eventually(r, time.Minute, time.Millisecond, func(tb assert.TB) {
				assert.True(tb, true, "the attempt passes")
			}, "the body settles")

			got := decoded(t, r.Records())
			if len(got) != 2 || got[0]["assertion"] != "eventually" || got[0]["seq"] != 1.0 {
				t.Fatalf("Records() = %v, want the call of eventually as 1 and its attempt's call", got)
			}
			if got[1]["assertion"] != "true" || got[1]["parent"] != 1.0 || got[1]["run"] != 1.0 {
				t.Fatalf("the second record is %v, want the attempt's call under 1 in run 1", got[1])
			}
		})

		t.Run("writes no record into the test's output while the switch is on", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				assert.True(assert.NewRecorder(), true, "the call on the recorder")
				return
			}
			out, err := childtest.Run(t, t.Name(), record.Variable+"=1")
			if err != nil || !strings.Contains(out, "--- PASS: "+t.Name()+" ") {
				t.Fatalf("the child exits with %v, want a pass:\n%s", err, out)
			}
			if strings.Contains(out, "the call on the recorder") {
				t.Fatalf("the child wrote the record of the recorder's call:\n%s", out)
			}
		})
	})
}

// The example uses a Recorder in place of *testing.T, so it prints the
// outcome instead of failing.
func ExampleNewRecorder() {
	r := assert.NewRecorder()
	fmt.Println(r.Failed())
	// Output: false
}

// TestRecorderClock checks the clock and the failure records of a
// recorder.
func TestRecorderClock(t *testing.T) {
	t.Parallel()

	t.Run("Clock", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the runtime clock by default", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			if got := r.Clock().Now(); got.Before(epoch) {
				t.Fatalf("Now() = %v, want a runtime reading", got)
			}
		})

		t.Run("returns the clock that WithClock set", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder().WithClock(assert.NewControlled(epoch))
			if got := r.Clock().Now(); !got.Equal(epoch) {
				t.Fatalf("Now() = %v, want the controlled clock's %v", got, epoch)
			}
		})
	})

	t.Run("Failures", func(t *testing.T) {
		t.Parallel()

		t.Run("returns every record in call order", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Equal(r, 1, 2, "the values match")

			records := r.Failures()
			if len(records) != 1 {
				t.Fatalf("Failures() returns %d records, want 1", len(records))
			}
			if got, want := records[0].Assertion, "equal"; got != want {
				t.Fatalf("Assertion = %q, want %q", got, want)
			}
		})

		t.Run("returns nothing for a message reported without a record", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("a bare message")

			if got := len(r.Failures()); got != 0 {
				t.Fatalf("Failures() returns %d records, want none", got)
			}
		})
	})
}

// TestRecorderContext checks the context of a recorder.
func TestRecorderContext(t *testing.T) {
	t.Parallel()

	t.Run("Context", func(t *testing.T) {
		t.Parallel()

		t.Run("returns context.Background() by default", func(t *testing.T) {
			t.Parallel()

			want := context.Background() //nolint:usetesting // a new Recorder returns it from Context
			if got := assert.NewRecorder().Context(); got != want {
				t.Fatalf("Context() = %v, want context.Background()", got)
			}
		})

		t.Run("returns the context that WithContext set", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(t.Context(), ledgerKey{}, "ledger")
			if got := assert.NewRecorder().WithContext(ctx).Context(); got != ctx {
				t.Fatalf("Context() = %v, want the context that WithContext set", got)
			}
		})
	})
}

// TestRecorderAllocs checks the allocation ceiling of NewRecorder and of
// each method of a recorder.
func TestRecorderAllocs(t *testing.T) {
	alloctest.Check(t, recorderCases())
}

// BenchmarkRecorder measures NewRecorder and each method of a recorder.
func BenchmarkRecorder(b *testing.B) {
	for _, c := range recorderCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// recorderCases returns a call of NewRecorder and of each method of a
// recorder that a failure of Equal reached, with its allocation ceiling,
// measured. The recorder keeps the first fatal message, so a later Fatalf
// formats none.
func recorderCases() []alloctest.Case {
	full := assert.NewRecorder()
	assert.Equal(full, 1, 2, "the values match")
	goexits := assert.NewRecorder()
	f := assert.Failure{Assertion: "true", Contract: allocContract}
	clock := assert.NewControlled(epoch)
	ctx := context.Background()
	return []alloctest.Case{
		{Name: "NewRecorder", Call: func(assert.TB) { recorder = assert.NewRecorder() }, Allocs: 1},
		{Name: "WithGoexit", Call: func(assert.TB) { goexits.WithGoexit() }},
		{Name: "Report", Call: func(assert.TB) { full.Report(f, false) }, Allocs: 2},
		{Name: "Failures", Call: func(assert.TB) { failures = full.Failures() }, Allocs: 1},
		{Name: "Records", Call: func(assert.TB) { lines = full.Records() }, Allocs: 1},
		{Name: "Clock", Call: func(assert.TB) { recorderClock = full.Clock() }},
		{Name: "WithClock", Call: func(assert.TB) { full.WithClock(clock) }},
		{Name: "Context", Call: func(assert.TB) { recorderContext = full.Context() }},
		{Name: "WithContext", Call: func(assert.TB) { full.WithContext(ctx) }},
		{Name: "Helper", Call: func(assert.TB) { full.Helper() }},
		{Name: "Fatalf", Call: func(assert.TB) { full.Fatalf("the flag is set") }},
		{Name: "Errorf", Call: func(assert.TB) { full.Errorf("the flag is set") }, Allocs: 1},
		{Name: "Failed", Call: func(assert.TB) { failed = full.Failed() }},
		{Name: "Message", Call: func(assert.TB) { message = full.Message() }},
		{Name: "Messages", Call: func(assert.TB) { messages = full.Messages() }, Allocs: 1},
		{Name: "HelperCalls", Call: func(assert.TB) { helpers = full.HelperCalls() }},
	}
}
