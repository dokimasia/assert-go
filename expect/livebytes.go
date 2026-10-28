// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// MaxLiveBytes records a failure and lets the test continue when fn keeps
// more than ceiling bytes of the heap reachable at a call of probe. It
// calls fn once with a probe that reads nothing to warm it. A run then
// collects and reads the live heap, and calls fn with a probe that collects
// and reads the live heap at each call. The excess of a run is its largest
// reading less the first reading. When the excess exceeds ceiling,
// MaxLiveBytes runs again, ten runs at most. The failure has the ceiling as
// want, the smallest excess as got, and the calls of the probe in the run
// of that excess as probes.
//
//	expect.MaxLiveBytes(t, func(probe func()) {
//	    for e := range filter(entries) {
//	        probe()
//	        sink = e
//	    }
//	}, 64<<10, "the filter keeps at most 64 KiB of a prefix")
//
// The caller puts the probe where the contract applies, such as before the
// read of each entry. A buffer that fn frees before its next probe is not
// read. A fn that calls the probe 0 times in its first run ends the call
// with a fault.
//
// A reading is the HeapAlloc of [runtime.ReadMemStats] after [runtime.GC],
// and costs a full collection. Every build checks the ceiling.
//
// # Parallel tests
//
// It reads the heap of the whole process, so the test that calls
// MaxLiveBytes does not call t.Parallel.
//
// # Allocation contract
//
// Besides what the calls of fn allocate, a passing call allocates the probe
// of each run.
func MaxLiveBytes(tb assert.TB, fn func(probe func()), ceiling uint64, msg string) {
	tb.Helper()
	matcher.MaxLiveBytes(tb, matcher.Soft, fn, ceiling, msg)
}
