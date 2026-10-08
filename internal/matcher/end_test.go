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

// fatalSeat is a seat that takes a fault as text. It keeps each message of
// Fatalf and Errorf, and counts the frames that mark themselves.
type fatalSeat struct {
	mu      sync.Mutex
	fatals  []string
	errs    []string
	helpers int
}

// Helper counts one mark.
func (s *fatalSeat) Helper() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.helpers++
}

// Fatalf keeps the message, and returns.
func (s *fatalSeat) Fatalf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fatals = append(s.fatals, fmt.Sprintf(format, args...))
}

// Errorf keeps the message.
func (s *fatalSeat) Errorf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, fmt.Sprintf(format, args...))
}

// TestEnd checks where the fault of a call that reports no verdict arrives.
func TestEnd(t *testing.T) {
	t.Parallel()

	t.Run("End", func(t *testing.T) {
		t.Parallel()

		t.Run("passes the fault to a FaultReporter as a fault that ends the call", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			err := fault.In("files.Workspace", fault.New("the tree cannot be written"))
			matcher.End(seat, err)
			if faults := seat.Faults(); len(faults) != 1 || !errors.Is(faults[0], err) {
				t.Fatalf("received the faults %v, want %v", faults, err)
			}
			if want := []string{matcher.RenderFault(err)}; !slices.Equal(seat.Fatals(), want) {
				t.Fatalf("the seat failed with %q, want %q from a fault that ends the call", seat.Fatals(), want)
			}
		})
		t.Run("sends the writer's text of the fault to any other seat through Fatalf", func(t *testing.T) {
			t.Parallel()

			seat := &fatalSeat{}
			err := fault.In("files.Workspace", fault.New("the tree cannot be written"))
			matcher.End(seat, err)
			if want := []string{matcher.RenderFault(err)}; !slices.Equal(seat.fatals, want) || len(seat.errs) != 0 {
				t.Fatalf("reported %q through Fatalf and %q through Errorf, want %q through Fatalf",
					seat.fatals, seat.errs, want)
			}
		})
		t.Run("marks its own frame as a helper of the seat", func(t *testing.T) {
			t.Parallel()

			seat := &fatalSeat{}
			matcher.End(seat, fault.New("the tree cannot be written"))
			if seat.helpers != 1 {
				t.Fatalf("marked %d frames, want 1", seat.helpers)
			}
		})
	})
}

// TestEndAllocs checks the allocation ceiling of End on a seat that takes a
// fault as the error it is.
func TestEndAllocs(t *testing.T) {
	checkAllocs(t, endCases())
}

// BenchmarkEnd measures End on a seat that takes a fault as the error it is.
func BenchmarkEnd(b *testing.B) {
	benchAllocs(b, endCases())
}

// endCases returns a call of End on a seat of internal/matchertest, which
// keeps the fault and the writer's text of it, with its allocation ceiling:
// the 3 allocations of the seat's ReportFault, which End adds nothing to.
func endCases() []allocCase {
	err := fault.In("files.Workspace", fault.New("the tree cannot be written"))
	return []allocCase{
		{name: "End", call: func(seat matcher.Seat) { matcher.End(seat, err) }, allocs: 4, fails: true},
	}
}
