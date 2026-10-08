// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// recovered keeps what a call of Panics returns.
var recovered any

// TestPanics runs the shared cases of the panic assertions.
func TestPanics(t *testing.T) {
	t.Parallel()

	t.Run("Panics", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPanics(t, func(s *matchertest.Seat, fn func(), msg string) any {
			return expect.Panics(s, fn, msg)
		})
	})

	t.Run("NotPanics", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNotPanics(t, func(s *matchertest.Seat, fn func(), msg string) {
			expect.NotPanics(s, fn, msg)
		})
	})
}

// TestPanicsAllocs checks the allocation ceiling of a passing call of each
// panic assertion.
func TestPanicsAllocs(t *testing.T) {
	alloctest.Check(t, panicsCases())
}

// BenchmarkPanics measures a passing call of each panic assertion.
func BenchmarkPanics(b *testing.B) {
	for _, c := range panicsCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// panicsCases returns a passing call of each panic assertion, with its
// allocation ceiling, measured. The subject of Panics panics with a
// constant text.
func panicsCases() []alloctest.Case {
	panicking := func() { panic("the key is empty") }
	return []alloctest.Case{
		{Name: "Panics", Call: func(tb assert.TB) { recovered = expect.Panics(tb, panicking, allocContract) }},
		{Name: "NotPanics", Call: func(tb assert.TB) { expect.NotPanics(tb, func() {}, allocContract) }},
	}
}
