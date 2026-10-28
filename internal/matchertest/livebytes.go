// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"strings"
	"testing"
)

// MaxLiveBytesInvoke calls a surface's ceiling on the live heap.
type MaxLiveBytesInvoke func(seat *Seat, fn func(probe func()), ceiling uint64, msg string)

// The filters of the cases read entries of entrySize bytes each.
const (
	// entries is the number of entries that a filter reads.
	entries = 256
	// entrySize is the size of one entry.
	entrySize = 1 << 10
	// liveCeiling is the ceiling of the cases: a quarter of the entries.
	liveCeiling = entries * entrySize / 4
	// warmAndTen is the number of times the assertion calls a callable over
	// its ceiling: once to warm it and once in each of ten runs.
	warmAndTen = 11
)

// sink keeps the entry that a filter wrote last, so escape analysis cannot
// remove its copy, and copied counts the bytes of the copies.
var (
	sink   []byte
	copied int
)

// input is the entries that the filters read.
var input = func() [][]byte {
	in := make([][]byte, entries)
	for i := range in {
		in[i] = make([]byte, entrySize)
	}
	return in
}()

// stream copies each entry and writes it before it calls probe for the next
// one, so it keeps one entry reachable at each probe.
func stream(probe func()) {
	for _, e := range input {
		probe()
		sink = append([]byte(nil), e...)
		copied += len(sink)
	}
	sink = nil
}

// buffer copies every entry before it writes the first one, so it keeps
// every entry read so far reachable at each probe.
func buffer(probe func()) {
	kept := make([][]byte, 0, entries)
	for _, e := range input {
		probe()
		kept = append(kept, append([]byte(nil), e...))
	}
	for _, e := range kept {
		sink = e
	}
	sink = nil
}

// RunMaxLiveBytes drives invoke against every case a ceiling on the live
// heap must produce.
//
// The readings cover the heap of the whole process, so the cases do not run
// in parallel, and the test that calls RunMaxLiveBytes does not call
// t.Parallel either.
func RunMaxLiveBytes(t *testing.T, invoke MaxLiveBytesInvoke) {
	t.Helper()

	t.Run("a callable that keeps one entry at each probe passes", func(t *testing.T) {
		copied = 0

		seat := &Seat{}
		invoke(seat, stream, liveCeiling, contractMsg)
		checkOutcome(t, seat, Case{})

		// A fixture that copies nothing keeps nothing, so the case
		// requires the copies.
		if copied == 0 {
			t.Fatal("the fixture copied no entry, so the case read no heap")
		}
	})

	t.Run("a callable that keeps every entry fails with the ceiling and the probes", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, buffer, liveCeiling, contractMsg)
		checkOutcome(t, seat, Case{
			Fails: true, Assertion: "max-live-bytes",
			Detail: map[string]any{"want": uint64(liveCeiling), "probes": uint64(entries)},
		})
		records := seat.Records()
		if len(records) == 0 {
			t.Fatal("the call reported no failure, so it read no excess")
		}
		if got, _ := records[0].Detail["got"].(uint64); got <= liveCeiling {
			t.Fatalf("got is %d bytes, want more than the ceiling of %d", got, liveCeiling)
		}
	})

	t.Run("a buffer that a callable frees before its next probe is not read", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, func(probe func()) {
			sink = make([]byte, 4*liveCeiling)
			sink = nil
			probe()
		}, liveCeiling, contractMsg)
		checkOutcome(t, seat, Case{})
	})

	t.Run("the call that warms the callable is not read", func(t *testing.T) {
		var cache []byte
		warmsOnce := func(probe func()) {
			if cache == nil {
				cache = make([]byte, 4*liveCeiling)
			}
			probe()
		}

		seat := &Seat{}
		invoke(seat, warmsOnce, liveCeiling, contractMsg)
		checkOutcome(t, seat, Case{})
		if cache == nil {
			t.Fatal("the fixture built no cache, so the case read no warm call")
		}
	})

	t.Run("a callable that never calls the probe ends the call with a fault", func(t *testing.T) {
		seat := &Seat{}
		invoke(seat, func(func()) {}, liveCeiling, contractMsg)
		faults := seat.Faults()
		if len(faults) != 1 || !strings.Contains(faults[0].Error(), "the callable called the probe 0 times") {
			t.Fatalf("the faults are %v, want the fault of a callable that called the probe 0 times", faults)
		}
	})

	t.Run("it calls a callable that meets its ceiling twice", func(t *testing.T) {
		calls := 0

		seat := &Seat{}
		invoke(seat, func(probe func()) {
			calls++
			probe()
		}, liveCeiling, contractMsg)

		if calls != 2 {
			t.Fatalf("called the callable %d times, want 2", calls)
		}
	})

	t.Run("it runs a callable over its ceiling ten times after the call that warms it", func(t *testing.T) {
		calls := 0

		seat := &Seat{}
		invoke(seat, func(probe func()) {
			calls++
			buffer(probe)
		}, liveCeiling, contractMsg)

		if calls != warmAndTen {
			t.Fatalf("called the callable over its ceiling %d times, want %d", calls, warmAndTen)
		}
	})
}
