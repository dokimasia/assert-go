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
// A contract is checked only when benchmarks run. An allocation ceiling
// that the ordinary test run checks is [go.dokimi.dev/assert.MaxAllocs].
//
// # Builds that allocate differently
//
// A build with the race detector, msan or asan allocates on the code's
// behalf, and under the race detector sync.Pool drops a quarter of the
// items it is given. A build whose -gcflags turn off optimisation or
// inlining, as a debugger's build does, moves values to the heap that an
// ordinary build keeps on the stack. In either build [Contract.End]
// publishes the allocation and byte counts and checks neither ceiling.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert for the seat,
// go.dokimi.dev/assert/expect for the ceilings, internal/matcher for the
// build's allocation flag, and the standard library's runtime, slices
// and time.
package bench
