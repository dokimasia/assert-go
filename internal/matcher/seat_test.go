// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"context"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The public seat satisfies this one by method set alone, which is
// what lets neither package import the other for its interface.
var _ matcher.Seat = assert.TB(nil)

// seatContext keeps the context that a call of ContextOf returns.
var seatContext any

// contextWrapper is a seat that wraps another seat and states a context of
// its own.
type contextWrapper struct {
	wrapper

	ctx context.Context //nolint:containedctx // the seat has a context, as testing.T has
}

// Context returns the wrapper's own context.
func (s contextWrapper) Context() context.Context { return s.ctx }

func TestSeat(t *testing.T) {
	t.Parallel()

	t.Run("Mode", func(t *testing.T) {
		t.Parallel()

		t.Run("the zero value is Fatal", func(t *testing.T) {
			t.Parallel()

			var zero matcher.Mode
			if zero != matcher.Fatal {
				t.Fatalf("the zero Mode is %v, want Fatal", zero)
			}
		})
	})

	t.Run("ContextOf", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the context that a seat states", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(t.Context(), ledgerKey{}, "ledger")
			if got := matcher.ContextOf(&contextSeat{ctx: ctx}); got != ctx {
				t.Fatalf("ContextOf returned %v, want the seat's context", got)
			}
		})

		t.Run("returns context.Background() for a seat without a context", func(t *testing.T) {
			t.Parallel()

			want := context.Background() //nolint:usetesting // ContextOf returns it for a seat without a context
			if got := matcher.ContextOf(&matchertest.Seat{}); got != want {
				t.Fatalf("ContextOf returned %v, want context.Background()", got)
			}
		})

		t.Run("returns context.Background() for a seat whose context is nil", func(t *testing.T) {
			t.Parallel()

			want := context.Background() //nolint:usetesting // ContextOf returns it for a seat whose context is nil
			if got := matcher.ContextOf(&contextSeat{}); got != want {
				t.Fatalf("ContextOf returned %v, want context.Background()", got)
			}
		})

		t.Run("returns the context of the seat that a seat wraps", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(t.Context(), ledgerKey{}, "ledger")
			if got := matcher.ContextOf(wrapper{wrapper{&contextSeat{ctx: ctx}}}); got != ctx {
				t.Fatalf("ContextOf returned %v, want the context of the wrapped seat", got)
			}
		})

		t.Run("returns the context of a wrapper that states its own", func(t *testing.T) {
			t.Parallel()

			inner := context.WithValue(t.Context(), ledgerKey{}, "inner")
			outer := context.WithValue(t.Context(), ledgerKey{}, "outer")
			seat := contextWrapper{wrapper: wrapper{&contextSeat{ctx: inner}}, ctx: outer}
			if got := matcher.ContextOf(seat); got != outer {
				t.Fatalf("ContextOf returned %v, want the wrapper's own context", got)
			}
		})

		t.Run("returns context.Background() for a seat whose Unwrap returns nil", func(t *testing.T) {
			t.Parallel()

			want := context.Background() //nolint:usetesting // ContextOf returns it for a chain without a context
			if got := matcher.ContextOf(wrapper{}); got != want {
				t.Fatalf("ContextOf returned %v, want context.Background()", got)
			}
		})
	})

	t.Run("Outer", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give func(seat matcher.Seat, body func(matcher.Seat))
		}{
			{
				name: "returns the seat of Rejects from the seat of its check",
				give: func(seat matcher.Seat, body func(matcher.Seat)) {
					matcher.Rejects(seat, matcher.Soft, "the check fails", func(s matcher.Seat) {
						body(s)
						s.Errorf("the check fails")
					})
				},
			},
			{
				name: "returns the seat of Eventually from the seat of an attempt",
				give: func(seat matcher.Seat, body func(matcher.Seat)) {
					matcher.Eventually(seat, matcher.Soft, time.Minute, time.Millisecond, body, "the attempt passes")
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				seat := &matchertest.Seat{}
				var got matcher.Seat
				tt.give(seat, func(s matcher.Seat) {
					if o, ok := s.(interface{ Outer() matcher.Seat }); ok {
						got = o.Outer()
					}
				})
				if got != matcher.Seat(seat) {
					t.Fatalf("Outer returned %v, want the seat of the assertion", got)
				}
			})
		}
	})
}

// TestSeatAllocs checks the allocation ceiling of each function of seat.go.
func TestSeatAllocs(t *testing.T) {
	checkAllocs(t, seatCases())
}

// BenchmarkSeat measures each function of seat.go.
func BenchmarkSeat(b *testing.B) {
	benchAllocs(b, seatCases())
}

// seatCases returns a call of each function of seat.go, with its allocation
// ceiling.
func seatCases() []allocCase {
	seat := &contextSeat{ctx: context.Background()}
	return []allocCase{
		{name: "ContextOf", call: func(matcher.Seat) { seatContext = matcher.ContextOf(seat) }},
	}
}
