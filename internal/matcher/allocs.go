// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"runtime/debug"
	"strings"
	"sync"
	"testing"
)

// allocRuns is the number of calls that [MaxAllocs] counts, after the
// call that warms the callable.
const allocRuns = 100

// gcflagsSetting is the key under which the build information records
// the -gcflags the binary was compiled with.
const gcflagsSetting = "-gcflags"

// The compiler flags that change what an ordinary build allocates: -N
// turns off optimisation and -l turns off inlining.
const (
	flagNoOptimisation = "-N"
	flagNoInlining     = "-l"
)

// MaxAllocs calls fn once to warm it, counts the heap allocations of
// the next 100 calls, and reports when their average, rounded down,
// exceeds ceiling. It counts as [testing.AllocsPerRun] counts.
//
//	matcher.MaxAllocs(seat, matcher.Fatal, func() { _, _ = store.Get(ctx, id) }, 0,
//	    "Get allocates nothing once the store is warm")
//
// In a build where [AllocationsCounted] reports false, it calls fn as
// an ordinary build does and reports nothing.
//
// testing.AllocsPerRun panics while a parallel test runs, so the test
// that calls MaxAllocs does not call t.Parallel.
func MaxAllocs(seat Seat, mode Mode, fn func(), ceiling uint64, msg string) {
	seat.Helper()

	got := uint64(testing.AllocsPerRun(allocRuns, fn))
	if AllocationsCounted() && got > ceiling {
		Fail(seat, mode, "max-allocs", msg, map[string]any{"want": ceiling, "got": got})
	}
}

// AllocationsCounted reports whether the running binary's allocation
// counts describe the code under test. They do not in two kinds of
// build:
//
//   - A build with the race detector, msan or asan, which allocates on
//     the code's behalf. Under the race detector sync.Pool also drops a
//     quarter of the items it is given.
//   - A build whose -gcflags turn off optimisation or inlining, as a
//     debugger's build does. With inlining off, a value that an ordinary
//     build keeps on the stack can move to the heap.
//
// It reads the build information once, and answers from that reading
// afterwards.
func AllocationsCounted() bool {
	return allocationsCounted()
}

// allocationsCounted computes [AllocationsCounted] on its first call.
var allocationsCounted = sync.OnceValue(func() bool {
	info, _ := debug.ReadBuildInfo()
	return !instrumented && !OptimisationsOff(info)
})

// OptimisationsOff reports whether info records -gcflags that turn off
// optimisation (-N) or inlining (-l) for any package. A flag may carry a
// package pattern, as in all=-N. A nil info, or one that records no
// -gcflags, reports false.
func OptimisationsOff(info *debug.BuildInfo) bool {
	if info == nil {
		return false
	}
	for _, setting := range info.Settings {
		if setting.Key == gcflagsSetting && disablesOptimisation(setting.Value) {
			return true
		}
	}
	return false
}

// disablesOptimisation reports whether flags, a -gcflags value, holds -N
// or -l. Each field is a flag, optionally after a package pattern and an
// equals sign, and the flag's name is compared whole, so -lang is not -l.
func disablesOptimisation(flags string) bool {
	for field := range strings.FieldsSeq(flags) {
		if pattern, flag, ok := strings.Cut(field, "="); ok && !strings.HasPrefix(pattern, "-") {
			field = flag
		}
		name, _, _ := strings.Cut(field, "=")
		if name == flagNoOptimisation || name == flagNoInlining {
			return true
		}
	}
	return false
}
