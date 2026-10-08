// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
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
			return matcher.Panics(s, matcher.Fatal, fn, msg)
		})
	})

	t.Run("NotPanics", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNotPanics(t, func(s *matchertest.Seat, fn func(), msg string) {
			matcher.NotPanics(s, matcher.Fatal, fn, msg)
		})
	})
}

// TestPanicsAllocs checks the allocation ceiling of a passing call of each
// panic assertion.
func TestPanicsAllocs(t *testing.T) {
	checkAllocs(t, panicsCases())
}

// BenchmarkPanics measures a passing call of each panic assertion.
func BenchmarkPanics(b *testing.B) {
	benchAllocs(b, panicsCases())
}

// panicsCases returns a passing call of each panic assertion, with its
// allocation ceiling. The subject of Panics panics with a constant text.
func panicsCases() []allocCase {
	panicking := func() { panic("the key is empty") }
	return []allocCase{
		{name: "Panics", call: func(seat matcher.Seat) {
			recovered = matcher.Panics(seat, matcher.Fatal, panicking, allocContract)
		}},
		{name: "NotPanics", call: func(seat matcher.Seat) {
			matcher.NotPanics(seat, matcher.Fatal, func() {}, allocContract)
		}},
	}
}
