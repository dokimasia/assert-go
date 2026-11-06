// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"slices"
	"testing"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// droppingSeat is a seat whose Skipf drops the reason, so that a call of Skip
// on it allocates only what Skip allocates.
type droppingSeat struct {
	matchertest.Seat
}

// Skipf drops the reason, and returns.
func (*droppingSeat) Skipf(string, ...any) {}

func TestSkip(t *testing.T) {
	t.Parallel()

	t.Run("Skip", func(t *testing.T) {
		t.Parallel()

		t.Run("calls Skipf of the seat with format and args", func(t *testing.T) {
			t.Parallel()

			seat := &skippingSeat{}
			matcher.Skip(seat, "assert.Skipf", "the stack at %s does not run", "nats")
			if want := []string{"the stack at nats does not run"}; !slices.Equal(seat.skips, want) || seat.Failed() {
				t.Fatalf("Skipf received %q and the seat %q, want %q through Skipf alone", seat.skips, seat.First(),
					want)
			}
		})

		t.Run("calls Skipf of the first seat of the chain that has the method", func(t *testing.T) {
			t.Parallel()

			seat := &skippingSeat{}
			matcher.Skip(wrapper{wrapper{seat}}, "assert.Skipf", "the stack does not run")
			if want := []string{"the stack does not run"}; !slices.Equal(seat.skips, want) {
				t.Fatalf("the wrapped seat received the skips %q, want %q", seat.skips, want)
			}
		})

		t.Run("ends the call with a fault of op on a chain without Skipf", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Skip(wrapper{seat}, "assert.Skipf", "the stack does not run")
			faults := seat.Faults()
			var got *fault.Error
			if len(faults) != 1 || !errors.As(faults[0], &got) || got.Op != "assert.Skipf" ||
				got.Reason != "the seat has no Skipf" {

				t.Fatalf("the seat received the faults %v, want the fault of assert.Skipf", faults)
			}
		})
	})
}

// TestSkipAllocs checks the allocation ceiling of Skip.
func TestSkipAllocs(t *testing.T) {
	checkAllocs(t, skipCases())
}

// BenchmarkSkip measures Skip.
func BenchmarkSkip(b *testing.B) {
	benchAllocs(b, skipCases())
}

// skipCases returns a call of Skip on a seat whose Skipf drops the reason,
// with its allocation ceiling.
func skipCases() []allocCase {
	seat := &droppingSeat{}
	return []allocCase{
		{name: "Skip", call: func(matcher.Seat) { matcher.Skip(seat, "assert.Skipf", "the stack does not run") }},
	}
}
