// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The child that TestBehaviour runs to watch a subject panic after
// CompletesWithin has stopped waiting for it.
const (
	// latePanic is the value the subject panics with.
	latePanic = "the subject panicked after the deadline"
	// childWait is how long the child waits for the panic after
	// CompletesWithin returns. The subject panics 50 milliseconds after
	// the deadline.
	childWait = 2 * time.Second
)

// TestCompletesWithinChild runs only in the child process. Its subject
// panics after the deadline, when CompletesWithin has returned, and the
// panic ends the process because nothing recovers it.
func TestCompletesWithinChild(t *testing.T) {
	if !childtest.InChild(t) {
		t.Skip("runs only as the child of TestBehaviour")
	}

	matcher.CompletesWithin(&matchertest.Seat{}, matcher.Soft, time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond)
		panic(latePanic)
	}, "the subject finishes in time")
	time.Sleep(childWait)
}

// TestBehaviour runs the shared cases of the behaviour assertions, and the
// cases of CompletesWithin and HonoursDeadline under a seat's clock.
func TestBehaviour(t *testing.T) {
	t.Parallel()

	t.Run("HonoursCancellation", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursCancellation(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.HonoursCancellation(s, matcher.Fatal, fn, msg)
		})
	})

	t.Run("HonoursDeadline", func(t *testing.T) {
		t.Parallel()
		matchertest.RunHonoursDeadline(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.HonoursDeadline(s, matcher.Fatal, fn, msg)
		})
	})

	t.Run("CompletesWithin", func(t *testing.T) {
		t.Parallel()
		matchertest.RunCompletesWithin(t, func(s *matchertest.Seat, within time.Duration,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.CompletesWithin(s, matcher.Fatal, within, fn, msg)
		})
	})

	t.Run("CompletesWithin passes a subject that takes the whole duration on the seat's clock", func(t *testing.T) {
		t.Parallel()

		clock := matcher.NewControlled(clockEpoch)
		seat := &clockedSeat{clock: clock}
		matcher.CompletesWithin(seat, matcher.Fatal, time.Minute, func(context.Context) error {
			clock.Advance(time.Minute)
			return nil
		}, "the subject finishes in time")

		if seat.Failed() {
			t.Fatalf("reported %q for a subject that took exactly the duration", seat.First())
		}
	})

	t.Run("CompletesWithin reports the time a subject took on the seat's clock", func(t *testing.T) {
		t.Parallel()

		clock := matcher.NewControlled(clockEpoch)
		seat := &clockedSeat{clock: clock}
		matcher.CompletesWithin(seat, matcher.Fatal, time.Minute, func(context.Context) error {
			clock.Advance(time.Minute + time.Second)
			return nil
		}, "the subject finishes in time")

		records := seat.Records()
		if len(records) != 1 {
			t.Fatalf("reported %d records, want 1", len(records))
		}
		want := map[string]any{"want": time.Minute, "got": time.Minute + time.Second}
		if got := records[0].Detail; got["want"] != want["want"] || got["got"] != want["got"] {
			t.Fatalf("the record states %v, want %v", got, want)
		}
	})

	t.Run("CompletesWithin panics on the subject's goroutine with a panic after the deadline", func(t *testing.T) {
		t.Parallel()

		out, err := childtest.Run(t, "TestCompletesWithinChild")
		if err == nil || !strings.Contains(out, "panic: "+latePanic) {
			t.Fatalf("the child exited with %v, want a crash on the panic %q:\n%s", err, latePanic, out)
		}
	})

	// A context decides expiry against the runtime clock and takes no
	// other. A deadline built from a seat clock ahead of the runtime has not
	// passed, and a subject that reports nothing would be reported as wrong.
	t.Run("HonoursDeadline reads the deadline off the runtime clock under a seat's clock", func(t *testing.T) {
		t.Parallel()

		honours := func(ctx context.Context) error { return ctx.Err() }
		tests := []struct {
			name string
			give time.Time
		}{
			{name: "passes a subject that honours its deadline under a clock behind the runtime's", give: clockEpoch},
			{
				name: "passes a subject that honours its deadline under a clock ahead of the runtime's",
				give: time.Now().Add(time.Hour),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				seat := &clockedSeat{clock: matcher.NewControlled(tt.give)}
				matcher.HonoursDeadline(seat, matcher.Fatal, honours, "the subject reports why it stopped")

				if seat.Failed() {
					t.Error("reported a subject that honoured its deadline")
				}
			})
		}
	})

	t.Run("Pure", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPure(t, func(s *matchertest.Seat, observe func() []int,
			fn func(), msg string,
		) {
			matcher.Pure(s, matcher.Fatal, observe, fn, msg)
		})
	})

	t.Run("NilContextSafe", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNilContextSafe(t, func(s *matchertest.Seat,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.NilContextSafe(s, matcher.Fatal, fn, msg)
		})
	})
}

// TestBehaviourAbandoned reads the goroutines of the whole process, so it
// does not run in parallel. Its leak check waits until the goroutine of the
// subject has ended.
func TestBehaviourAbandoned(t *testing.T) {
	t.Run("CompletesWithin ends the goroutine of a subject that returns after its deadline", func(t *testing.T) {
		check := matcher.NoGoroutineLeaks(t, matcher.Fatal, "the subject's goroutine ends")
		release := make(chan struct{})
		seat := &matchertest.Seat{}
		matcher.CompletesWithin(seat, matcher.Soft, time.Millisecond, func(context.Context) error {
			<-release
			return nil
		}, "the subject finishes in time")
		close(release)
		check()

		if records := seat.Records(); len(records) != 1 || records[0].Assertion != "completes-within" {
			t.Fatalf("reported %+v, want one record of completes-within", records)
		}
	})
}

// TestBehaviourAllocs checks the allocation ceiling of a passing call of
// each behaviour assertion.
func TestBehaviourAllocs(t *testing.T) {
	checkAllocs(t, behaviourCases())
}

// BenchmarkBehaviour measures a passing call of each behaviour assertion.
func BenchmarkBehaviour(b *testing.B) {
	benchAllocs(b, behaviourCases())
}

// behaviourCases returns a passing call of each behaviour assertion, with
// its allocation ceiling, measured. Each subject returns at once.
func behaviourCases() []allocCase {
	honours := func(ctx context.Context) error { return ctx.Err() }
	quick := func(context.Context) error { return nil }
	observe := func() int { return 1 }
	return []allocCase{
		{name: "HonoursCancellation", allocs: 2, call: func(seat matcher.Seat) {
			matcher.HonoursCancellation(seat, matcher.Fatal, honours, allocContract)
		}},
		{name: "HonoursDeadline", allocs: 2, call: func(seat matcher.Seat) {
			matcher.HonoursDeadline(seat, matcher.Fatal, honours, allocContract)
		}},
		{name: "CompletesWithin", allocs: 14, call: func(seat matcher.Seat) {
			matcher.CompletesWithin(seat, matcher.Fatal, time.Minute, quick, allocContract)
		}},
		{name: "Pure", call: func(seat matcher.Seat) {
			matcher.Pure(seat, matcher.Fatal, observe, func() {}, allocContract)
		}},
		{name: "NilContextSafe", call: func(seat matcher.Seat) {
			matcher.NilContextSafe(seat, matcher.Fatal, quick, allocContract)
		}},
	}
}
