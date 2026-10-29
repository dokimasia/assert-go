// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/record"
)

// The results of the allocation cases, which keep each call.
var (
	recorder        *assert.Recorder
	failures        []assert.Failure
	faults          []error
	lines           []string
	messages        []string
	message         string
	failed          bool
	helpers         int
	recorderClock   assert.Clock
	recorderContext any
	recorderTest    assert.TB
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

	t.Run("ReportFault", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a fault that ends its call and fails the recorder with its text", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.ReportFault(errMissingOption, true)

			if kept := r.Faults(); len(kept) != 1 || !errors.Is(kept[0], errMissingOption) {
				t.Fatalf("Faults() = %v, want %v", kept, errMissingOption)
			}
			if !r.Failed() || r.Message() != matcher.RenderFault(errMissingOption) {
				t.Fatalf("Failed() = %v and Message() = %q, want a failure with the writer's text of the fault",
					r.Failed(), r.Message())
			}
		})

		t.Run("keeps a noted fault without failing the recorder", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.ReportFault(errMissingOption, false)

			if kept := r.Faults(); len(kept) != 1 || !errors.Is(kept[0], errMissingOption) {
				t.Fatalf("Faults() = %v, want %v", kept, errMissingOption)
			}
			if r.Failed() {
				t.Fatalf("Failed() = true with the message %q, want no failure for a noted fault", r.Message())
			}
		})
	})

	t.Run("Faults", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nothing for a recorder that received no fault", func(t *testing.T) {
			t.Parallel()

			if got := assert.NewRecorder().Faults(); len(got) != 0 {
				t.Fatalf("Faults() = %v, want none", got)
			}
		})

		t.Run("returns every fault in call order", func(t *testing.T) {
			t.Parallel()

			second := errors.New("the store cannot write its entry")
			r := assert.NewRecorder()
			r.ReportFault(second, false)
			r.ReportFault(errMissingOption, true)

			if got := r.Faults(); len(got) != 2 || !errors.Is(got[0], second) || !errors.Is(got[1], errMissingOption) {
				t.Fatalf("Faults() = %v, want the noted fault and then the fault that ended the call", got)
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

// TestRecorderTest checks the test that a recorder runs in.
func TestRecorderTest(t *testing.T) {
	t.Parallel()

	t.Run("Test", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil by default", func(t *testing.T) {
			t.Parallel()

			if got := assert.NewRecorder().Test(); got != nil {
				t.Fatalf("Test() = %v, want nil", got)
			}
		})

		t.Run("returns the test that WithTest set", func(t *testing.T) {
			t.Parallel()

			if got := assert.NewRecorder().WithTest(t).Test(); got != assert.TB(t) {
				t.Fatalf("Test() = %v, want the test that WithTest set", got)
			}
		})
	})
}

// TestRecorderCleanups checks the cleanups that a recorder keeps and runs.
func TestRecorderCleanups(t *testing.T) {
	t.Parallel()

	t.Run("Cleanup", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps f until RunCleanups", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			ran := false
			r.Cleanup(func() { ran = true })

			if ran {
				t.Fatal("Cleanup ran f, want f kept until RunCleanups")
			}
		})
	})

	t.Run("RunCleanups", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the functions that Cleanup registered, the last registered first", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			var steps []string
			assert.Cleanup(r, func() { steps = append(steps, "first") })
			r.Cleanup(func() { steps = append(steps, "second") })
			r.RunCleanups()

			if got, want := fmt.Sprint(steps), "[second first]"; got != want {
				t.Fatalf("RunCleanups ran %s, want %s", got, want)
			}
		})

		t.Run("leaves a function that arrives after it to the next call", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			var steps []string
			r.Cleanup(func() { steps = append(steps, "first run") })
			r.RunCleanups()
			r.Cleanup(func() { steps = append(steps, "second run") })

			if got, want := fmt.Sprint(steps), "[first run]"; got != want {
				t.Fatalf("one call ran %s, want %s", got, want)
			}
			r.RunCleanups()
			if got, want := fmt.Sprint(steps), "[first run second run]"; got != want {
				t.Fatalf("two calls ran %s, want %s", got, want)
			}
		})

		t.Run("runs the earlier functions when a function ends the goroutine under WithGoexit", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder().WithGoexit()
			var steps []string
			r.Cleanup(func() { steps = append(steps, "first") })
			r.Cleanup(func() {
				steps = append(steps, "second")
				assert.True(r, false, "the release fails")
			})
			done := make(chan struct{})
			go func() {
				defer close(done)
				r.RunCleanups()
			}()
			<-done

			if got, want := fmt.Sprint(steps), "[second first]"; got != want || r.Message() == "" {
				t.Fatalf("RunCleanups ran %s, and recorded %q, want %s and the failure of the release",
					got, r.Message(), want)
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
// recorder that a failing Equal calls, with its allocation ceiling. The
// recorder keeps the first fatal message, so a later Fatalf formats none.
func recorderCases() []alloctest.Case {
	full := assert.NewRecorder()
	assert.Equal(full, 1, 2, "the values match")
	goexits := assert.NewRecorder()
	cleaning := assert.NewRecorder()
	f := assert.Failure{Assertion: "true", Contract: allocContract}
	clock := assert.NewControlled(epoch)
	ctx := context.Background()
	return []alloctest.Case{
		{Name: "NewRecorder", Call: func(assert.TB) { recorder = assert.NewRecorder() }, Allocs: 2},
		{Name: "WithGoexit", Call: func(assert.TB) { goexits.WithGoexit() }},
		{Name: "Report", Call: func(assert.TB) { full.Report(f, false) }, Allocs: 3},
		{Name: "ReportFault", Call: func(assert.TB) { full.ReportFault(errMissingOption, false) }, Allocs: 2},
		{Name: "Failures", Call: func(assert.TB) { failures = full.Failures() }, Allocs: 2},
		{Name: "Faults", Call: func(assert.TB) { faults = full.Faults() }, Allocs: 2},
		{Name: "Records", Call: func(assert.TB) { lines = full.Records() }, Allocs: 2},
		{Name: "Clock", Call: func(assert.TB) { recorderClock = full.Clock() }},
		{Name: "WithClock", Call: func(assert.TB) { full.WithClock(clock) }},
		{Name: "Context", Call: func(assert.TB) { recorderContext = full.Context() }},
		{Name: "WithContext", Call: func(assert.TB) { full.WithContext(ctx) }},
		{Name: "Test", Call: func(assert.TB) { recorderTest = full.Test() }},
		{Name: "WithTest", Call: func(tb assert.TB) { full.WithTest(tb) }},
		{Name: "Helper", Call: func(assert.TB) { full.Helper() }},
		{Name: "Fatalf", Call: func(assert.TB) { full.Fatalf("the flag is set") }},
		{Name: "Errorf", Call: func(assert.TB) { full.Errorf("the flag is set") }, Allocs: 2},
		{Name: "Failed", Call: func(assert.TB) { failed = full.Failed() }},
		{Name: "Message", Call: func(assert.TB) { message = full.Message() }},
		{Name: "Messages", Call: func(assert.TB) { messages = full.Messages() }, Allocs: 2},
		{Name: "HelperCalls", Call: func(assert.TB) { helpers = full.HelperCalls() }},
		{Name: "Cleanup", Call: func(assert.TB) {
			cleaning.Cleanup(noCleanup)
			cleaning.RunCleanups()
		}},
		{Name: "RunCleanups", Call: func(assert.TB) { cleaning.RunCleanups() }},
	}
}
