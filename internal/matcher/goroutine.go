// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"bytes"
	"errors"
	"runtime"
	"slices"
	"strconv"
	"time"

	"go.dokimi.dev/assert/internal/fault"
)

// The bounds of the stack dump that a leak check reads.
const (
	// firstStackDump is the size of the first buffer that a leak check
	// reads the dump into: 1 MiB.
	firstStackDump = 1 << 20
	// maxStackDump is the size of the largest: 64 MiB, the bound that
	// runtime/pprof sets on its own dump of every goroutine.
	maxStackDump = 64 << 20
	// maxDumps bounds the dumps of one call to [DumpAll]. A buffer that
	// starts empty and doubles at each dump is 2^62 bytes at the 64th.
	maxDumps = 64
)

// The wait of a leak check for goroutines that are about to return: up
// to 100 readings, 5 ms apart, which span half a second.
const (
	leakReadings               = 100
	leakInterval time.Duration = 5_000_000
)

// ErrDumpTooLarge is the error of a dump that fills the largest buffer
// that [DumpAll] may use.
var ErrDumpTooLarge = errors.New("matcher: the dump fills the largest buffer")

// NoGoroutineLeaks records which goroutines are running and returns a
// check that reports the goroutines started after this call that are
// still running when the check is called.
//
//	done := matcher.NoGoroutineLeaks(seat, matcher.Fatal, "the worker stops")
//	defer done()
//
// The check compares goroutine ids, not counts, so a goroutine that was
// already running is never reported.
//
// The check reads the running goroutines up to 100 times, 5 ms apart, and
// reports the new goroutines that are still running at the last reading.
// A goroutine that returns during that half second is not a leak. The
// wait is real time, because no clock that a test controls affects when a
// goroutine returns.
//
// Each reading dumps the stacks of every goroutine into a buffer of at
// most 64 MiB. A dump that does not fit leaves the set of new goroutines
// unknown, so the check ends without a verdict, with a fault of the kind
// [ErrDumpTooLarge].
//
// # Parallel tests
//
// A goroutine that a parallel test starts between the two readings is
// new, so the check reports it as a leak. Do not call [testing.T.Parallel]
// in a test that uses this check. A package whose other tests are
// parallel can still produce a false report, because each reading covers
// the whole process.
//
// # Allocation contract
//
// A call and a check that finds no new goroutine allocate 7 times in a
// process of a few goroutines, the 1 MiB buffer of the dumps among them.
// The sets of goroutine ids grow with the goroutines of the process, so a
// process of more goroutines allocates more.
func NoGoroutineLeaks(seat Seat, mode Mode, msg string) func() {
	seat.Helper()

	before, buf, err := goroutineIDs(make([]byte, firstStackDump))
	return func() {
		seat.Helper()

		var leaked []uint64
		if err == nil {
			leaked, err = stillRunning(before, buf)
		}
		if err != nil {
			Fault(seat, mode, "no-task-leaks", msg, fault.Of(err,
				"the stacks of every goroutine fill %d bytes, which leaves the new goroutines unknown", maxStackDump))
			return
		}
		if len(leaked) > 0 {
			Fail(seat, mode, "no-task-leaks", msg, map[string]any{"leaked": leaked})
			return
		}
		Pass(seat, mode, "no-task-leaks", msg)
	}
}

// stillRunning reads the running goroutines up to 100 times, 5 ms apart,
// into buf, and returns the goroutines that are not in before and still
// run at the last reading, in ascending order. It returns none as soon as
// a reading finds none.
func stillRunning(before map[uint64]bool, buf []byte) ([]uint64, error) {
	var leaked []uint64
	for range leakReadings {
		running, next, err := goroutineIDs(buf)
		if err != nil {
			return nil, err
		}
		buf = next
		if leaked = newIDs(before, running); len(leaked) == 0 {
			return nil, nil
		}
		time.Sleep(leakInterval)
	}
	return leaked, nil
}

// allStacks writes the stack of every goroutine into b, and returns the
// number of bytes it wrote.
func allStacks(b []byte) int { return runtime.Stack(b, true) }

// goroutineIDs returns the id of every goroutine in the stacks of every
// goroutine, and the buffer that it read them into. It reads into buf, and
// grows it up to 64 MiB when the stacks do not fit.
//
// A caller that reads in a loop passes the returned buffer back, so the
// loop allocates once rather than once per reading.
func goroutineIDs(buf []byte) (map[uint64]bool, []byte, error) {
	buf, n, err := DumpAll(buf, maxStackDump, allStacks)
	if err != nil {
		return nil, nil, err
	}

	out := map[uint64]bool{}
	for line := range bytes.Lines(buf[:n]) {
		if id, ok := goroutineID(line); ok {
			out[id] = true
		}
	}
	return out, buf, nil
}

// DumpAll returns a buffer that contains the whole of a dump, and the
// dump's length. dump writes into the buffer it is given and returns the
// number of bytes it wrote, as runtime.Stack does.
//
// A dump that fills its buffer may have been cut short, so DumpAll dumps
// again into a buffer twice as large, up to a buffer of limit bytes. It
// returns [ErrDumpTooLarge] when a dump fills the buffer of limit bytes.
//
// # Allocation contract
//
// DumpAll allocates nothing for a dump that fits in buf, and a buffer for
// each dump that fills the one before it.
func DumpAll(buf []byte, limit int, dump func([]byte) int) ([]byte, int, error) {
	for range maxDumps {
		if n := dump(buf); n < len(buf) {
			return buf, n, nil
		}
		if len(buf) == limit {
			break
		}
		buf = make([]byte, min(max(2*len(buf), 1), limit))
	}
	return nil, 0, ErrDumpTooLarge
}

// goroutineID reads the id from a "goroutine 12 [running]:" header,
// reporting false for any other line of a dump.
func goroutineID(line []byte) (uint64, bool) {
	rest, ok := bytes.CutPrefix(line, []byte("goroutine "))
	if !ok {
		return 0, false
	}
	digits, _, _ := bytes.Cut(rest, []byte(" "))
	id, err := strconv.ParseUint(string(digits), 10, 64)
	return id, err == nil
}

// newIDs returns the ids in after that are not in before, in ascending
// order so a failure reads the same way twice.
func newIDs(before, after map[uint64]bool) []uint64 {
	var out []uint64
	for id := range after {
		if !before[id] {
			out = append(out, id)
		}
	}

	slices.Sort(out)
	return out
}
