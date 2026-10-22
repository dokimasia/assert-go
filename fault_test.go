// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matcher"
)

// errMissingOption is the fault of a helper that misses an option.
var errMissingOption = errors.New("the check calls Restart without the option Restartable")

// textSeat is a seat that takes a fault as text. It keeps each message of
// Fatalf and Errorf.
type textSeat struct {
	mu     sync.Mutex
	fatals []string
	errs   []string
}

// Helper does nothing.
func (*textSeat) Helper() {}

// Fatalf keeps the message, and returns.
func (s *textSeat) Fatalf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fatals = append(s.fatals, fmt.Sprintf(format, args...))
}

// Errorf keeps the message.
func (s *textSeat) Errorf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, fmt.Sprintf(format, args...))
}

// handle is the seat of a check of a helper library, which wraps the seat of
// its test.
type handle struct {
	assert.TB
}

// Unwrap returns the seat that the handle wraps.
func (h handle) Unwrap() assert.TB { return h.TB }

// TestFault checks where the fault of a helper arrives.
func TestFault(t *testing.T) {
	t.Parallel()

	t.Run("Fault", func(t *testing.T) {
		t.Parallel()

		t.Run("passes the fault to a recorder as a fault that ends the call", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Fault(r, errMissingOption)
			if faults := r.Faults(); len(faults) != 1 || !errors.Is(faults[0], errMissingOption) {
				t.Fatalf("the recorder received the faults %v, want %v", faults, errMissingOption)
			}
			if !r.Failed() || r.Message() != matcher.RenderFault(errMissingOption) {
				t.Fatalf("the recorder failed: %v, with %q, want a failure with the writer's text of the fault",
					r.Failed(), r.Message())
			}
		})
		t.Run("sends the writer's text of the fault through Fatalf of a seat that takes no fault", func(t *testing.T) {
			t.Parallel()

			seat := &textSeat{}
			assert.Fault(seat, errMissingOption)
			if want := []string{matcher.RenderFault(errMissingOption)}; !slices.Equal(seat.fatals, want) ||
				len(seat.errs) != 0 {

				t.Fatalf("reported %q through Fatalf and %q through Errorf, want %q through Fatalf",
					seat.fatals, seat.errs, want)
			}
		})
		t.Run("passes the fault to the recorder that a handle wraps", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Fault(handle{r}, errMissingOption)
			if faults := r.Faults(); len(faults) != 1 || !errors.Is(faults[0], errMissingOption) {
				t.Fatalf("the wrapped recorder received the faults %v, want %v", faults, errMissingOption)
			}
		})
		t.Run("writes no call record", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Fault(r, errMissingOption)
			if records := r.Records(); len(records) != 0 {
				t.Fatalf("the recorder keeps the call records %q, want none", records)
			}
		})
		t.Run("panics for a nil fault", func(t *testing.T) {
			t.Parallel()

			defer func() {
				if got := recover(); got != "assert: Fault(tb, nil) states no fault" {
					t.Fatalf("panicked with %v, want the panic of a nil fault", got)
				}
			}()
			assert.Fault(assert.NewRecorder(), nil)
		})
	})
}

// TestFaultAllocs checks the allocation ceiling of Fault on a recorder.
func TestFaultAllocs(t *testing.T) {
	alloctest.Check(t, faultCases())
}

// BenchmarkFault measures Fault on a recorder.
func BenchmarkFault(b *testing.B) {
	for _, c := range faultCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// faultCases returns a call of Fault on a recorder, which keeps the fault and
// its text, with its allocation ceiling.
func faultCases() []alloctest.Case {
	r := assert.NewRecorder()
	return []alloctest.Case{
		{Name: "Fault", Call: func(assert.TB) { assert.Fault(r, errMissingOption) }, Allocs: 2},
	}
}
