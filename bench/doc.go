// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package bench fails a benchmark that exceeds a stated ceiling.
//
// On its own, a benchmark prints measurements and never fails on them. A
// contract states a ceiling in the benchmark, and a run that exceeds the
// ceiling fails the build:
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
// Set a ceiling from a measurement, above the run-to-run noise. A ceiling
// at the measured value fails on a run that is slower by chance. A
// ceiling far above it never fails, and it gives a false guarantee.
//
// # Failure semantics
//
// [Contract.End] checks every stated ceiling and fails the benchmark for
// each one exceeded. It reports the ceilings together, so one run names
// every ceiling that was exceeded.
//
// # What is measured where
//
// Latency and allocation are measured differently. Only latency is
// available in every language implementing this standard. See
// [Contract.MaxLatency] and [Contract.MaxAllocs].
//
// A contract is checked only when benchmarks run. An allocation ceiling
// that the ordinary test run checks is [go.dokimi.dev/assert.MaxAllocs].
//
// # Builds that allocate differently
//
// A build with the race detector, msan or asan allocates on the code's
// behalf. Under the race detector, sync.Pool also drops a quarter of the
// items it is given. A build whose -gcflags turn off optimisation or
// inlining, as a debugger's build does, moves values to the heap that an
// ordinary build keeps on the stack. In either build [Contract.End]
// publishes the allocation and byte counts and leaves both ceilings
// unchecked.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert for the seat, internal/matcher for the
// verdicts of the ceilings and the build's allocation flag, and the
// standard library's flag, fmt, math, runtime, slices, strconv, strings,
// sync, sync/atomic, testing and time.
package bench
