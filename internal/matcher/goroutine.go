// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"bytes"
	"context"
	"math/rand/v2"
	"runtime/pprof"
	"slices"
	"strconv"
	"time"
)

// scopeLabel is the key of the profiler label that marks the goroutines
// that a leak check's scope starts.
const scopeLabel = "dokimi.assert.scope"

// The wait of a leak check for goroutines that are about to return: up
// to 100 readings, 5 ms apart, which span half a second.
const (
	leakReadings               = 100
	leakInterval time.Duration = 5_000_000
)

// profileBytes is the size of the buffer that a leak check reads the
// goroutine profile into, which the profile of a test process of a few
// dozen goroutines does not outgrow: 64 KiB.
const profileBytes = 64 << 10

// NoGoroutineLeaks marks the calling goroutine with a profiler label of a
// scope of its own, and returns a check that reports the goroutines with
// that label that are still running when the check is called.
//
//	done := matcher.NoGoroutineLeaks(seat, matcher.Fatal, "the worker stops")
//	defer done()
//
// A goroutine inherits the labels of the goroutine that starts it, so every
// goroutine that the scope starts has the label, and so does every
// goroutine that one of those starts. A goroutine of another test never has
// it, so the check reports no goroutine that a parallel test starts. The
// check clears the labels of the goroutine that calls it, which is the
// goroutine that called NoGoroutineLeaks, so that goroutine is not
// reported.
//
// The check reads the goroutine profile up to 100 times, 5 ms apart, and
// reports the labelled goroutines that are still running at the last
// reading. A goroutine that returns during that half second is not a leak.
// The wait is real time, because no clock that a test controls affects when
// a goroutine returns. The detail leaked states the function that each
// leaked goroutine runs, its outermost frame, in ascending order.
//
// # Limits
//
// The check has three limits:
//
//   - A goroutine that a goroutine started before the call starts on the
//     scope's behalf, such as a worker of a pool, does not have the label.
//   - Code that sets labels from a context without the scope's label, such as
//     [pprof.Do] with a context of its own, removes the label from the
//     goroutines that it starts.
//   - The call replaces the labels that the calling goroutine had, and the
//     check clears them, so labels that the caller set before the call are
//     gone after the check.
//
// # Allocation contract
//
// A call and a check that finds no labelled goroutine allocate 163 times in
// a process of a few goroutines: the label, the 64 KiB buffer of the
// profile, and what the goroutine profile allocates for each group of
// goroutines. The profile grows with the goroutines of the process.
func NoGoroutineLeaks(seat Seat, mode Mode, msg string) func() {
	seat.Helper()

	value := strconv.FormatUint(rand.Uint64(), 36)
	pprof.SetGoroutineLabels(pprof.WithLabels(context.Background(), pprof.Labels(scopeLabel, value)))
	return func() {
		seat.Helper()

		pprof.SetGoroutineLabels(context.Background())
		if leaked := stillRunning([]byte(`"` + scopeLabel + `":"` + value + `"`)); len(leaked) > 0 {
			Fail(seat, mode, "no-task-leaks", msg, map[string]any{"leaked": leaked})
			return
		}
		Pass(seat, mode, "no-task-leaks", msg)
	}
}

// stillRunning reads the goroutine profile up to 100 times, 5 ms apart, and
// returns the function of each goroutine whose labels contain label that
// still runs at the last reading, in ascending order. It returns none as
// soon as a reading finds none.
func stillRunning(label []byte) []string {
	b := bytes.NewBuffer(make([]byte, 0, profileBytes))
	var leaked []string
	for range leakReadings {
		b.Reset()
		// A goroutine profile writes into a buffer without an error.
		_ = pprof.Lookup("goroutine").WriteTo(b, 1)
		if leaked = labelled(b.Bytes(), label); len(leaked) == 0 {
			return nil
		}
		time.Sleep(leakInterval)
	}
	slices.Sort(leaked)
	return leaked
}

// labelled returns the function of each goroutine of profile, a goroutine
// profile at debug level 1, whose labels contain label. A group of n
// goroutines of one stack and one label set states n of them.
//
// A group is a header "n @ addresses", the line "# labels: {…}" of a group
// with labels, and a line "#\taddress\tfunction+offset\tfile:line" for each
// frame, innermost first, which ends at the function that the goroutine
// runs. The profile aligns the columns of a frame with tabs.
func labelled(profile, label []byte) []string {
	var out []string
	var function []byte
	count, marked := 0, false
	flush := func() {
		if marked {
			name := string(function)
			for range count {
				out = append(out, name)
			}
		}
		count, marked, function = 0, false, nil
	}
	for line := range bytes.Lines(profile) {
		if head, _, isHeader := bytes.Cut(line, []byte(" @ ")); isHeader {
			flush()
			count, _ = strconv.Atoi(string(head))
		} else if bytes.HasPrefix(line, []byte("# labels: ")) {
			marked = bytes.Contains(line, label)
		} else if frame, isFrame := bytes.CutPrefix(line, []byte("#\t")); isFrame {
			_, named, _ := bytes.Cut(frame, []byte("\t"))
			function, _, _ = bytes.Cut(bytes.TrimLeft(named, "\t"), []byte("+"))
		}
	}
	flush()
	return out
}
