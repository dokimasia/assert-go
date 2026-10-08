// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package alloctest checks the allocation ceiling of each function of a
// package from one table of cases: in the Allocs test that go test runs,
// and in the benchmark whose contract states the same ceiling.
//
// A case is one passing call of a function on a seat that writes no call
// record, with the ceiling of its allocations.
// [go.dokimi.dev/assert/expect.MaxAllocs] counts the allocations of a call
// as their average, rounded to the nearest whole number. The ceiling is 0
// for a call that allocates nothing. For any other call it is a quarter
// above the highest count that Linux, macOS and Windows measure, rounded up
// to two significant digits.
//
//	func truthCases() []alloctest.Case {
//	    return []alloctest.Case{
//	        {Name: "True", Call: func(tb assert.TB) { assert.True(tb, true, "the flag is set") }},
//	    }
//	}
//
//	func TestTruthAllocs(t *testing.T) { alloctest.Check(t, truthCases()) }
//
//	func BenchmarkTruth(b *testing.B) {
//	    for _, c := range truthCases() {
//	        b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
//	    }
//	}
//
// # Dependency position
//
// Imports expect, which checks a ceiling in a test, bench, whose contract
// checks it in a benchmark, assert, which declares the seat and checks the
// verdict of a benchmark's calls, and internal/matchertest, whose seat
// writes no call record. Only the test files of this module import it.
package alloctest
