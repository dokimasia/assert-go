// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"bytes"
	"runtime"
	"slices"
	"strconv"
	"time"
)

// Buffer sizes for the stack dump a leak check reads. It grows rather
// than guessing, because a truncated dump loses the goroutines at the
// end and would report them as finished.
const (
	firstStackBuf = 1 << 20
	maxStackBuf   = 8 << 20
)

// How long a leak check waits for goroutines to finish before
// reporting them. Real time, and necessarily: no clock a test controls
// governs when a goroutine returns.
const (
	leakGrace    = 500 * time.Millisecond
	leakInterval = 5 * time.Millisecond
)

// NoGoroutineLeaks records which goroutines are running and returns a
// check that reports any still running when it is called.
//
//	done := matcher.NoGoroutineLeaks(seat, matcher.Fatal, "the worker stops")
//	defer done()
//
// Identity, not count: only goroutines started after this call are
// reported, so goroutines already running are never blamed.
//
// The check waits up to half a second, polling, before reporting. A
// goroutine on its way out is not a leak, and without the grace period
// every test that starts one would be flaky.
//
// # What it cannot tell apart
//
// A goroutine started by a neighbouring test between the two readings
// looks exactly like a leak, because both are simply new. Do not call
// [testing.T.Parallel] in a test using this, and be aware that a
// package whose other tests are parallel can still produce a false
// report. The reading is over the whole process; nothing scopes it to
// one test.
func NoGoroutineLeaks(seat Seat, mode Mode, msg string) func() {
	seat.Helper()

	before, _ := goroutineIDs(make([]byte, firstStackBuf))

	return func() {
		seat.Helper()

		deadline := time.Now().Add(leakGrace)
		var leaked []uint64
		buf := make([]byte, firstStackBuf)
		for {
			var running map[uint64]bool
			running, buf = goroutineIDs(buf)
			leaked = newIDs(before, running)
			if len(leaked) == 0 || time.Now().After(deadline) {
				break
			}
			time.Sleep(leakInterval)
		}

		if len(leaked) > 0 {
			Fail(seat, mode, "no-task-leaks", msg, map[string]any{"leaked": leaked})
		}
	}
}

// goroutineIDs returns the id of every goroutine running now, reading
// into buf and growing it when the dump does not fit.
//
// It answers the buffer it ended with so a caller polling in a loop
// allocates once rather than once per reading. Each reading stops the
// world, and the dump is a megabyte before it grows.
func goroutineIDs(buf []byte) (map[uint64]bool, []byte) {
	var n int
	for {
		n = runtime.Stack(buf, true)
		if n < len(buf) || len(buf) >= maxStackBuf {
			break
		}
		buf = make([]byte, min(len(buf)*2, maxStackBuf))
	}

	out := map[uint64]bool{}
	for line := range bytes.Lines(buf[:n]) {
		if id, ok := goroutineID(line); ok {
			out[id] = true
		}
	}
	return out, buf
}

// goroutineID reads the id from a "goroutine 12 [running]:" header,
// reporting false for any other line.
func goroutineID(line []byte) (uint64, bool) {
	header := []byte("goroutine ")

	if !bytes.HasPrefix(line, header) {
		return 0, false
	}
	digits, _, ok := bytes.Cut(line[len(header):], []byte(" "))
	if !ok {
		return 0, false
	}

	id, err := strconv.ParseUint(string(digits), 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
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
