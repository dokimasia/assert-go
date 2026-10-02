// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The child that TestBehaviour runs to watch a subject panic after
// CompletesWithin has stopped waiting for it.
const (
	// lateSubjectEnv makes TestCompletesWithinChild run the subject.
	lateSubjectEnv = "MATCHER_LATE_SUBJECT"
	// latePanic is the value the subject panics with.
	latePanic = "the subject panicked after the deadline"
	// childWait is how long the child waits for the panic after
	// CompletesWithin returns. The subject panics 50 milliseconds after
	// the deadline.
	childWait = 2 * time.Second
	// childDeadline bounds the child process.
	childDeadline = 20 * time.Second
)

// TestCompletesWithinChild runs only in the child process. Its subject
// panics after the deadline, when CompletesWithin has returned, and the
// panic ends the process because nothing recovers it.
func TestCompletesWithinChild(t *testing.T) {
	if os.Getenv(lateSubjectEnv) != "1" {
		t.Skip("runs only as the child of TestBehaviour")
	}

	matcher.CompletesWithin(&matchertest.Seat{}, matcher.Soft, time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond)
		panic(latePanic)
	}, "the subject finishes in time")
	time.Sleep(childWait)
}

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

		executable, err := os.Executable()
		if err != nil {
			t.Fatalf("the test binary's path: %v", err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), childDeadline)
		defer cancel()

		child := exec.CommandContext(ctx, executable,
			"-test.run=^TestCompletesWithinChild$", "-test.timeout="+childDeadline.String())
		child.Env = append(os.Environ(), lateSubjectEnv+"=1")
		out, err := child.CombinedOutput()

		if err == nil || !strings.Contains(string(out), "panic: "+latePanic) {
			t.Fatalf("the child exited with %v, want a crash on the panic %q:\n%s", err, latePanic, out)
		}
	})

	t.Run("CompletesWithin raises the panic of a subject that panics after its deadline", func(t *testing.T) {
		t.Parallel()

		if raised := matchertest.Raised(func() { matcher.EndAbandoned(latePanic) }); raised != latePanic {
			t.Fatalf("recovered %v, want the subject's own panic value %q", raised, latePanic)
		}
	})

	t.Run("CompletesWithin raises nothing for a subject that returns after its deadline", func(t *testing.T) {
		t.Parallel()

		if raised := matchertest.Raised(func() { matcher.EndAbandoned(nil) }); raised != nil {
			t.Fatalf("recovered %v from a subject that returned after its deadline, want nothing", raised)
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
