// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// logSeat is a seat with a log, as testing.T has one, which takes a fault
// as text. It records whether a failure arrived.
type logSeat struct {
	mu     sync.Mutex
	logs   []string
	failed bool
}

// Helper marks nothing: the seat states no location.
func (*logSeat) Helper() {}

// Fatalf records that a failure arrived, and returns.
func (s *logSeat) Fatalf(string, ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed = true
}

// Errorf records that a failure arrived.
func (s *logSeat) Errorf(string, ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed = true
}

// Logf keeps the formatted text.
func (s *logSeat) Logf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, fmt.Sprintf(format, args...))
}

// reportingLogSeat is a seat with a log that takes a fault as a value.
type reportingLogSeat struct {
	matchertest.Seat

	mu   sync.Mutex
	logs []string
}

// Logf keeps the formatted text.
func (s *reportingLogSeat) Logf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, fmt.Sprintf(format, args...))
}

// TestNote checks where a note and a fault that does not end its call
// arrive.
func TestNote(t *testing.T) {
	t.Parallel()

	t.Run("Note", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the text into the log of a seat with a log", func(t *testing.T) {
			t.Parallel()

			seat := &logSeat{}
			matcher.Note(seat, "the cart had 3 items")
			if want := []string{"the cart had 3 items"}; !slices.Equal(seat.logs, want) {
				t.Fatalf("logged %q, want %q", seat.logs, want)
			}
			if seat.failed {
				t.Fatal("reported a failure for a note")
			}
		})
		t.Run("writes nothing to a seat without a log", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Note(seat, "the cart had 3 items")
			if seat.Failed() || seat.HelperCalls() != 1 {
				t.Fatalf(
					"reported %q with %d helper marks, want nothing and one mark",
					seat.First(),
					seat.HelperCalls(),
				)
			}
		})
	})

	t.Run("NoteFault", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the writer's text of the fault into the log of a seat with a log", func(t *testing.T) {
			t.Parallel()

			seat := &logSeat{}
			err := fault.In("prop.ForAll", fault.New("the stored case decodes to other values"))
			matcher.NoteFault(seat, err)
			if want := []string{matcher.RenderFault(err)}; !slices.Equal(seat.logs, want) {
				t.Fatalf("logged %q, want %q", seat.logs, want)
			}
			if seat.failed {
				t.Fatal("reported a failure for a fault that does not end the call")
			}
		})
		t.Run("passes the fault to a FaultReporter as a fault that does not end the call", func(t *testing.T) {
			t.Parallel()

			seat := &reportingLogSeat{}
			err := fault.In("prop.ForAll", fault.New("the stored case decodes to other values"))
			matcher.NoteFault(seat, err)
			if faults := seat.Faults(); len(faults) != 1 || !errors.Is(faults[0], err) {
				t.Fatalf("received the faults %v, want %v", faults, err)
			}
			if len(seat.logs) != 0 || seat.Failed() {
				t.Fatalf("logged %q and reported %q, want neither for a fault that a seat takes",
					seat.logs, seat.First())
			}
		})
	})
}

// TestNoteAllocs checks the allocation ceiling of Note and of NoteFault on
// a seat without a log.
func TestNoteAllocs(t *testing.T) {
	checkAllocs(t, noteCases())
}

// BenchmarkNote measures Note and NoteFault on a seat without a log.
func BenchmarkNote(b *testing.B) {
	benchAllocs(b, noteCases())
}

// noteCases returns a call of Note and of NoteFault on a seat of
// internal/matchertest, which has no log and takes a fault as the error it
// is, with its allocation ceiling.
func noteCases() []allocCase {
	err := fault.In("prop.ForAll", fault.New("the stored case decodes to other values"))
	return []allocCase{
		{name: "Note", call: func(seat matcher.Seat) { matcher.Note(seat, "the cart had 3 items") }},
		{name: "NoteFault", call: func(seat matcher.Seat) { matcher.NoteFault(seat, err) }},
	}
}
