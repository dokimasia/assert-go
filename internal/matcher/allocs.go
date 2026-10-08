// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"flag"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
)

// allocRuns is the number of calls that [MaxAllocs] and
// [MaxAllocsWithSetup] count, after the call that warms the callable.
const allocRuns = 100

// testLogFlag is the flag of the testing package that names the test log,
// which go test passes to a run whose result it can cache.
const testLogFlag = "test.testlogfile"

// testLogNote ends the note of an allocation ceiling that only the test log
// leaves unchecked, after the id of the assertion.
const testLogNote = ": the ceiling is not checked under go test's test log, which allocates. Run -count=1 to check it."

// gcflagsSetting is the key under which the build information records
// the -gcflags the binary was compiled with.
const gcflagsSetting = "-gcflags"

// The compiler flags that change what an ordinary build allocates: -N
// turns off optimisation and -l turns off inlining.
const (
	flagNoOptimisation = "-N"
	flagNoInlining     = "-l"
)

// MaxAllocs calls fn once to warm it and counts the heap allocations of
// the next 100 calls. It reports when their average, rounded to the
// nearest whole number, exceeds ceiling. A half rounds up, so a callable
// that allocates on 50 of the 100 calls fails a ceiling of 0.
//
//	matcher.MaxAllocs(seat, matcher.Fatal, func() { _, _ = store.Get(ctx, id) }, 0,
//	    "Get allocates nothing per call once the store is warm")
//
// In a run where [AllocationsCounted] reports false, it calls fn as an
// ordinary build does and passes. Where only the test log of go test stops
// the check, it writes a note into the log of seat that a run with
// -count=1 checks the ceiling.
//
// It counts as [testing.AllocsPerRun] counts, with GOMAXPROCS at 1 and one
// reading of the counter before the 100 calls and one after them. The count
// covers the whole process, so the test that calls MaxAllocs does not call
// t.Parallel.
//
// # Allocation contract
//
// A passing call allocates nothing besides what the 101 calls of fn
// allocate, and the note that the Logf of seat writes.
func MaxAllocs(seat Seat, mode Mode, fn func(), ceiling uint64, msg string) {
	seat.Helper()

	got := allocs(fn)
	if ceilingChecked(seat, "max-allocs"+testLogNote) && got > ceiling {
		Fail(seat, mode, "max-allocs", msg, map[string]any{"want": ceiling, "got": got})
		return
	}
	Pass(seat, mode, "max-allocs", msg)
}

// MaxAllocsWithSetup calls setup, and fn on the input that setup returns,
// once to warm both. It then counts the heap allocations of the next 100
// calls of fn, each on an input that a call of setup builds outside the
// count. It reports when their average, rounded to the nearest whole
// number, exceeds ceiling. A half rounds up. It sets GOMAXPROCS to 1 while
// it counts, as [testing.AllocsPerRun] does.
//
//	matcher.MaxAllocsWithSetup(seat, matcher.Fatal, freshStore, (*Store).Settle, 4,
//	    "settling a store allocates at most four times")
//
// In a run where [AllocationsCounted] reports false, it calls setup and fn
// as an ordinary build does and passes. Where only the test log of go test
// stops the check, it writes a note into the log of seat, as [MaxAllocs]
// does.
//
// The count covers the whole process, so the test that calls
// MaxAllocsWithSetup does not call t.Parallel.
//
// # Allocation contract
//
// A passing call allocates nothing besides what the 101 calls of setup and
// of fn allocate, and the note that the Logf of seat writes.
func MaxAllocsWithSetup[T any](seat Seat, mode Mode, setup func() T, fn func(T), ceiling uint64, msg string) {
	seat.Helper()

	got := allocsAfter(setup, fn)
	if ceilingChecked(seat, "max-allocs-with-setup"+testLogNote) && got > ceiling {
		Fail(seat, mode, "max-allocs-with-setup", msg, map[string]any{"want": ceiling, "got": got})
		return
	}
	Pass(seat, mode, "max-allocs-with-setup", msg)
}

// allocs returns the heap allocations per call of fn over allocRuns calls
// after one call that warms it, as [perCall] rounds them. It reads the
// counter before the calls and after them.
func allocs(fn func()) uint64 {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))

	fn()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	before := stats.Mallocs
	for range allocRuns {
		fn()
	}
	runtime.ReadMemStats(&stats)
	return perCall(stats.Mallocs - before)
}

// allocsAfter returns the heap allocations per call of fn over allocRuns
// calls after one call that warms setup and fn, as [perCall] rounds them.
// Each counted call takes an input that a call of setup builds before the
// counter is read.
func allocsAfter[T any](setup func() T, fn func(T)) uint64 {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))

	fn(setup())
	var stats runtime.MemStats
	var total uint64
	for range allocRuns {
		input := setup()
		runtime.ReadMemStats(&stats)
		before := stats.Mallocs
		fn(input)
		runtime.ReadMemStats(&stats)
		total += stats.Mallocs - before
	}
	return perCall(total)
}

// perCall returns total, the heap allocations of allocRuns calls, as a count
// per call: the average rounded to the nearest whole number, with a half
// rounded up.
func perCall(total uint64) uint64 {
	return (2*total + allocRuns) / (2 * allocRuns)
}

// ceilingChecked reports whether an allocation ceiling applies, as
// [AllocationsCounted] reports it. Where the build counts allocations and
// only the test log stops the check, it writes note into the log of seat.
func ceilingChecked(seat Seat, note string) bool {
	seat.Helper()
	counted := AllocationsCounted()
	if !counted && builtToCount() {
		Note(seat, note)
	}
	return counted
}

// AllocationsCounted reports whether the running binary's allocation
// counts describe the code under test. They do not describe it in four
// kinds of run:
//
//   - A build with the race detector, msan or asan, which allocates on
//     the code's behalf. Under the race detector sync.Pool also drops a
//     quarter of the items it is given.
//   - A build whose -gcflags turn off optimisation or inlining, as a
//     debugger's build does. With inlining off, a value that an ordinary
//     build keeps on the stack can move to the heap.
//   - A test binary that a mutation run instrumented, as
//     [MutationInstrumented] reports. The binary contains every mutant of
//     a package behind a switch, so the compiler inlines fewer of its
//     functions, and a value can move to the heap as it does with inlining
//     off. The ordinary build of one mutant, which a mutation run builds to
//     confirm a survivor, counts its allocations.
//   - A run of a test binary that writes the test log of go test. go test
//     passes the flag -test.testlogfile to every run whose result it can
//     cache, and the log records the files that the run opens and stats.
//     An entry of an os.Root allocates, so on Go 1.27.1 an Open and a Close
//     through a Root allocate 6 times with the log and 5 times without it.
//     A run that turns the cache off, such as one with -count=1, writes no
//     log.
//
// It reads the build information and the environment on its first call,
// and returns the result of that reading afterwards. It reads the flag on
// every call, because a TestMain can call it before the testing package
// parses the flags.
//
// # Allocation contract
//
// AllocationsCounted allocates nothing after its first call.
func AllocationsCounted() bool {
	return builtToCount() && !testLogged()
}

// builtToCount reports whether the build and the environment of the running
// binary let its allocation counts describe the code under test, on its
// first call, and returns the result of that call afterwards.
var builtToCount = sync.OnceValue(func() bool {
	info, _ := debug.ReadBuildInfo()
	return !instrumented && !OptimisationsOff(info) && !MutationInstrumented()
})

// testLogged reports whether the running binary writes the test log of go
// test: whether its flag -test.testlogfile names a file. A binary without
// the testing package has no such flag.
func testLogged() bool {
	f := flag.Lookup(testLogFlag)
	//dokimi:mutate-skip ror-true,lcr-right: every test binary has the flag, so no test runs where f is nil
	return f != nil &&
		f.Value.String() != ""
}

// OptimisationsOff reports whether info records -gcflags that turn off
// optimisation (-N) or inlining (-l) for any package. A flag may follow a
// package pattern, as in all=-N. A nil info, or one that records no
// -gcflags, reports false.
//
// # Allocation contract
//
// OptimisationsOff allocates nothing.
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

// disablesOptimisation reports whether flags, a -gcflags value, contains
// -N or -l. Each field is a flag, optionally after a package pattern and an
// equals sign, and the flag's name is compared whole, so -lang is not -l.
func disablesOptimisation(flags string) bool {
	for field := range strings.FieldsSeq(flags) {
		if pattern, flag, _ := strings.Cut(field, "="); !strings.HasPrefix(pattern, "-") {
			field = flag
		}
		name, _, _ := strings.Cut(field, "=")
		if name == flagNoOptimisation || name == flagNoInlining {
			return true
		}
	}
	return false
}
