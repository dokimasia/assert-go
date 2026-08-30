// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package bench fails a benchmark that exceeds a stated ceiling.
//
// A benchmark on its own records numbers; somebody has to read them to
// notice a regression. A contract states the ceiling in the benchmark,
// so exceeding it fails the build instead:
//
//	func BenchmarkGet(b *testing.B) {
//	    c := bench.Start(b).MaxLatency(50 * time.Microsecond).MaxAllocs(2)
//	    defer c.End()
//
//	    for c.Loop() {
//	        _, _ = store.Get(ctx, id)
//	    }
//	}
//
// # Choosing a ceiling
//
// Set it from a measurement, above the noise. A ceiling at the current
// number fails on the first unlucky run; one far above it never fails
// at all. Neither is worth having, and the second is worse, because it
// reads as a guarantee.
//
// # Failure semantics
//
// Every ceiling is checked by [Contract.End], and exceeding one stops
// the benchmark. Ceilings are reported together, so one run names
// every ceiling exceeded rather than the first.
//
// # What is measured where
//
// Latency and allocation are measured differently, and only latency is
// available in every language implementing this standard. See
// [Contract.MaxLatency] and [Contract.MaxAllocs].
//
// # Dependency position
//
// Imports go.dokimi.dev/assert for the seat, and the standard
// library's fmt, runtime, slices, strings and time.
package bench
