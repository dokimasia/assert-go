// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"context"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
)

// Compile-time proof that both seats satisfy TB. A runtime nil check
// cannot state this: the compiler already knows the value is non-nil.
var (
	_ assert.TB = (*testing.T)(nil)
	_ assert.TB = (*testing.B)(nil)
	_ assert.TB = (*assert.Recorder)(nil)
)

// tbContext keeps the context that a call of Context returns.
var tbContext context.Context

// bareSeat is a seat with the three methods of TB and nothing else, as the
// seat of a generated check body can be.
type bareSeat struct{}

func (bareSeat) Helper()               {}
func (bareSeat) Fatalf(string, ...any) {}
func (bareSeat) Errorf(string, ...any) {}

func TestTB(t *testing.T) {
	t.Parallel()

	t.Run("Helper", func(t *testing.T) {
		t.Parallel()

		t.Run("reaches the seat through the interface", func(t *testing.T) {
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

		t.Run("reaches the seat through the interface", func(t *testing.T) {
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

		t.Run("reaches the seat through the interface", func(t *testing.T) {
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

		t.Run("returns the context that a recorder states", func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(t.Context(), ledgerKey{}, "ledger")
			if got := assert.Context(assert.NewRecorder().WithContext(ctx)); got != ctx {
				t.Fatalf("Context of the recorder = %v, want the context that WithContext set", got)
			}
		})

		t.Run("returns context.Background() for a seat without a context", func(t *testing.T) {
			t.Parallel()

			if got := assert.Context(bareSeat{}); got != context.Background() {
				t.Fatalf("Context of a bare seat = %v, want context.Background()", got)
			}
		})
	})
}

// TestTBAllocs checks the allocation ceiling of Context.
func TestTBAllocs(t *testing.T) {
	alloctest.Check(t, tbCases())
}

// BenchmarkTB measures Context.
func BenchmarkTB(b *testing.B) {
	for _, c := range tbCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// tbCases returns a call of Context on a recorder that states a context,
// with its allocation ceiling, measured.
func tbCases() []alloctest.Case {
	seat := assert.NewRecorder().WithContext(context.Background())
	return []alloctest.Case{
		{Name: "Context", Call: func(assert.TB) { tbContext = assert.Context(seat) }},
	}
}
