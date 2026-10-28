// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"math"
	"runtime"

	"go.dokimi.dev/assert/internal/fault"
)

// liveRuns is the most runs that [MaxLiveBytes] takes. The assertion fails
// when none of them meets the ceiling.
const liveRuns = 10

// MaxLiveBytes calls fn once with a probe that reads nothing to warm it. A
// run then collects and reads the live heap, and calls fn with a probe that
// collects and reads the live heap at each call. The excess of a run is its
// largest reading less the first reading, or 0. When the excess exceeds
// ceiling, MaxLiveBytes runs again, ten runs at most. When no run meets
// ceiling, it reports the smallest excess and the probes of its run.
//
//	matcher.MaxLiveBytes(seat, matcher.Fatal, func(probe func()) {
//	    for e := range filter(entries) {
//	        probe()
//	        sink = e
//	    }
//	}, 64<<10, "the filter keeps at most 64 KiB of a prefix")
//
// A fn that calls the probe 0 times in its first run ends the call with a
// fault, because no reading compared anything with the ceiling.
//
// A reading is the HeapAlloc of [runtime.ReadMemStats] after [runtime.GC].
// It covers the heap of the whole process, so the test that calls
// MaxLiveBytes does not call t.Parallel.
//
// # Allocation contract
//
// Besides what the calls of fn allocate, a passing call allocates the probe
// of each run and the record of the call.
func MaxLiveBytes(seat Seat, mode Mode, fn func(probe func()), ceiling uint64, msg string) {
	seat.Helper()

	fn(func() {})
	lowest, probes := uint64(math.MaxUint64), uint64(0)
	for run := range liveRuns {
		excess, calls := liveExcess(fn)
		if run == 0 && calls == 0 {
			Fault(seat, mode, "max-live-bytes", msg, fault.New("the callable called the probe 0 times"))
			return
		}
		if excess <= ceiling {
			Pass(seat, mode, "max-live-bytes", msg)
			return
		}
		if excess < lowest {
			lowest, probes = excess, calls
		}
	}
	Fail(seat, mode, "max-live-bytes", msg,
		map[string]any{wantField: ceiling, gotField: lowest, probesField: probes})
}

// liveExcess runs fn once and returns its excess, the largest reading of the
// live heap at a call of the probe less the reading before the call, and the
// calls of the probe. The probe is built before the first reading, so the
// reading contains it.
func liveExcess(fn func(probe func())) (excess, calls uint64) {
	var first uint64
	probe := func() {
		calls++
		if h := liveHeap(); h > first {
			excess = max(excess, h-first)
		}
	}
	first = liveHeap()
	fn(probe)
	return excess, calls
}

// liveHeap collects the heap and returns the bytes of its live objects.
func liveHeap() uint64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}
