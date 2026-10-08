// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"context"
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
)

// Compile-time proof that both seats of a test and the recorder satisfy TB.
var (
	_ expect.TB = (*testing.T)(nil)
	_ expect.TB = (*testing.B)(nil)
	_ expect.TB = (*expect.Recorder)(nil)
)

// tbContext keeps the context that a call of Context returns.
var tbContext any

func TestTB(t *testing.T) {
	t.Parallel()

	t.Run("TB", func(t *testing.T) {
		t.Parallel()

		t.Run("is the seat of the aborting surface", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.TB](), reflect.TypeFor[assert.TB](), "one seat for both surfaces")
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the context of the seat, as the aborting surface does", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, expect.Context(t), t.Context(), "the context of the test")
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
// with its allocation ceiling.
func tbCases() []alloctest.Case {
	seat := expect.NewRecorder().WithContext(context.Background())
	return []alloctest.Case{
		{Name: "Context", Call: func(assert.TB) { tbContext = expect.Context(seat) }},
	}
}
