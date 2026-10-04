// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package alloctest

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/matchertest"
)

// Case is a passing call of one function, and the ceiling of its
// allocations.
type Case struct {
	// Name is the name of the function, which names the case's
	// sub-benchmark.
	Name string
	// Call calls the function on tb, a seat that writes no call record.
	Call func(tb assert.TB)
	// Allocs is the ceiling: the allocations of one call, measured.
	Allocs uint64
}

// Check checks the ceiling of each case with [expect.MaxAllocs] on tb, and
// that each call passes. It calls each case on a seat of
// internal/matchertest of its own. It reports every case past its ceiling,
// so one run states each ceiling that changed.
//
// testing.AllocsPerRun, which it counts with, panics while a parallel test
// runs, so the test that calls it does not call t.Parallel.
func Check(tb assert.TB, cases []Case) {
	tb.Helper()
	for _, c := range cases {
		seat := &matchertest.Seat{}
		expect.MaxAllocs(tb, func() { c.Call(seat) }, c.Allocs, c.Name+" allocates within its ceiling")
		expect.False(tb, seat.Failed(), c.Name+" passes")
	}
}

// Measure measures c in the benchmark b, under a contract whose allocation
// ceiling is the ceiling of c, and checks that each call passes. It calls c
// on a seat of internal/matchertest, once before the contract starts, so the
// allocations of the setup of a first call count in no iteration, as they
// count in no call that Check counts.
func Measure(b bench.B, c Case) {
	b.Helper()
	seat := &matchertest.Seat{}
	c.Call(seat)
	contract := bench.Start(b).MaxAllocs(c.Allocs)
	defer contract.End()
	for contract.Loop() {
		c.Call(seat)
	}
	assert.False(b, seat.Failed(), c.Name+" passes")
}
