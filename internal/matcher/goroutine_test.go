// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// Not parallel, and neither are its cases: the reading is over the
// whole process. See [matchertest.RunNoGoroutineLeaks].
func TestGoroutine(t *testing.T) {
	t.Run("NoGoroutineLeaks", func(t *testing.T) {
		matchertest.RunNoGoroutineLeaks(t, func(s *matchertest.Seat, msg string) func() {
			return matcher.NoGoroutineLeaks(s, matcher.Fatal, msg)
		})
	})
}

// firstDump is the size of the first buffer that a leak check reads the
// stacks into. A limit of this size refuses a dump that fills that buffer.
const firstDump = 1 << 20

// leakMsg is the caller's message of every leak check below.
const leakMsg = "the worker stops"

// unreadableWords end the report of a leak check whose stacks do not fit.
const unreadableWords = "which leaves the set of new goroutines unknown"

// readings returns a dump that writes texts in turn, and the last text
// for every reading after them, with a count of the readings.
func readings(texts ...string) (func([]byte) int, *int) {
	calls := 0
	return func(b []byte) int {
		text := texts[min(calls, len(texts)-1)]
		calls++
		return copy(b, text)
	}, &calls
}

// fills is a dump that fills every buffer it is given.
func fills(b []byte) int { return len(b) }

// TestNoLeaks drives the leak check over stacks that the test writes, so
// no reading covers the process and the cases run in parallel.
func TestNoLeaks(t *testing.T) {
	t.Parallel()

	t.Run("reports the new goroutines still running at the last of 100 readings", func(t *testing.T) {
		t.Parallel()

		dump, calls := readings(
			"goroutine 1 [running]:\n",
			"goroutine 1 [running]:\n\ngoroutine 9 [select]:\n\ngoroutine 4 [chan receive]:\n",
		)
		seat := &matchertest.Seat{}
		matcher.NoLeaks(seat, matcher.Fatal, leakMsg, dump, firstDump)()

		records := seat.Records()
		if len(records) != 1 || records[0].Assertion != "no-task-leaks" {
			t.Fatalf("reported %+v, want one record of no-task-leaks", records)
		}
		leaked, _ := records[0].Detail["leaked"].([]uint64)
		if want := []uint64{4, 9}; !slices.Equal(leaked, want) {
			t.Errorf("leaked = %v, want %v in ascending order", records[0].Detail["leaked"], want)
		}
		if *calls != 101 {
			t.Errorf("read the stacks %d times, want the first reading and 100 more", *calls)
		}
	})

	t.Run("stops reading once no new goroutine is running", func(t *testing.T) {
		t.Parallel()

		dump, calls := readings(
			"goroutine 1 [running]:\n",
			"goroutine 1 [running]:\n\ngoroutine 7 [select]:\n",
			"goroutine 1 [running]:\n",
		)
		seat := &matchertest.Seat{}
		matcher.NoLeaks(seat, matcher.Fatal, leakMsg, dump, firstDump)()

		if seat.Failed() {
			t.Fatalf("reported %q for a goroutine that ended at the second reading", seat.First())
		}
		if *calls != 3 {
			t.Errorf("read the stacks %d times, want 3", *calls)
		}
	})

	t.Run("reads no id from a line that only starts like a header", func(t *testing.T) {
		t.Parallel()

		dump, calls := readings(
			"goroutine 1 [running]:\n",
			"goroutine 1 [running]:\ngoroutine 2\ngoroutine x [running]:\ngoroutine  [running]:\n",
		)
		seat := &matchertest.Seat{}
		matcher.NoLeaks(seat, matcher.Fatal, leakMsg, dump, firstDump)()

		if seat.Failed() {
			t.Fatalf("reported %q for lines that state no goroutine id", seat.First())
		}
		if *calls != 2 {
			t.Errorf("read the stacks %d times, want 2", *calls)
		}
	})

	// The later readings fit, so a check that went on from the first
	// reading would report every goroutine that they list.
	t.Run("reports stacks that fill the largest buffer at the first reading", func(t *testing.T) {
		t.Parallel()

		calls := 0
		dump := func(b []byte) int {
			calls++
			if calls == 1 {
				return len(b)
			}
			return copy(b, "goroutine 1 [running]:\n")
		}
		seat := &matchertest.Seat{}
		matcher.NoLeaks(seat, matcher.Fatal, leakMsg, dump, firstDump)()

		checkUnreadable(t, seat)
		if calls != 1 {
			t.Errorf("read the stacks %d times, want the first reading alone", calls)
		}
	})

	t.Run("reports stacks that outgrow the largest buffer at a later reading", func(t *testing.T) {
		t.Parallel()

		calls := 0
		dump := func(b []byte) int {
			calls++
			if calls == 1 {
				return copy(b, "goroutine 1 [running]:\n")
			}
			return len(b)
		}
		seat := &matchertest.Seat{}
		matcher.NoLeaks(seat, matcher.Fatal, leakMsg, dump, firstDump)()

		checkUnreadable(t, seat)
		if calls != 2 {
			t.Errorf("read the stacks %d times, want 2", calls)
		}
	})

	t.Run("reports through Errorf under Soft", func(t *testing.T) {
		t.Parallel()

		seat := &matchertest.Seat{}
		matcher.NoLeaks(seat, matcher.Soft, leakMsg, fills, firstDump)()

		if errs := seat.Errs(); len(errs) != 1 || len(seat.Fatals()) != 0 {
			t.Fatalf("reported %q through Errorf and %q through Fatalf, want one through Errorf",
				errs, seat.Fatals())
		}
	})
}

// checkUnreadable fails t unless seat took one aborting report, without a
// record, that the stacks did not fit.
func checkUnreadable(t *testing.T, seat *matchertest.Seat) {
	t.Helper()

	fatals := seat.Fatals()
	if len(fatals) != 1 || !strings.HasPrefix(fatals[0], leakMsg+": ") ||
		!strings.HasSuffix(fatals[0], unreadableWords) {
		t.Fatalf("reported %q, want one report that the stacks do not fit", fatals)
	}
	if records := seat.Records(); len(records) != 0 {
		t.Errorf("reported the records %+v for a check that states no verdict", records)
	}
}

// sized returns a dump of size bytes, and the sizes of the buffers that it
// was given, in order.
func sized(size int) (func([]byte) int, *[]int) {
	var given []int
	return func(b []byte) int {
		given = append(given, len(b))
		return min(len(b), size)
	}, &given
}

func TestDumpAll(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		first   int
		limit   int
		size    int
		buffers []int
	}{
		{
			name:  "dumps once into a buffer that the dump does not fill",
			first: 16, limit: 1024, size: 3, buffers: []int{16},
		},
		{
			name:  "doubles the buffer until a dump leaves room",
			first: 16, limit: 1024, size: 100, buffers: []int{16, 32, 64, 128},
		},
		{
			name:  "dumps again after a dump that fills its buffer exactly",
			first: 16, limit: 1024, size: 16, buffers: []int{16, 32},
		},
		{
			name:  "caps the last buffer at the limit",
			first: 16, limit: 100, size: 99, buffers: []int{16, 32, 64, 100},
		},
		{
			name:  "grows an empty buffer",
			first: 0, limit: 1024, size: 3, buffers: []int{0, 1, 2, 4},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dump, given := sized(tc.size)
			buf, n, err := matcher.DumpAll(make([]byte, tc.first), tc.limit, dump)
			if err != nil {
				t.Fatalf("DumpAll() = %v, want the whole dump", err)
			}
			if n != tc.size || len(buf) != tc.buffers[len(tc.buffers)-1] {
				t.Errorf("DumpAll() = %d bytes in a buffer of %d, want %d in the last buffer",
					n, len(buf), tc.size)
			}
			if !slices.Equal(*given, tc.buffers) {
				t.Errorf("dumped into buffers of %v bytes, want %v", *given, tc.buffers)
			}
		})
	}

	t.Run("refuses a dump that fills the buffer of limit bytes", func(t *testing.T) {
		t.Parallel()

		dump, given := sized(100)
		buf, _, err := matcher.DumpAll(make([]byte, 16), 100, dump)
		if !errors.Is(err, matcher.ErrDumpTooLarge) || buf != nil {
			t.Fatalf("DumpAll() = a buffer of %d bytes and %v, want no buffer and %v",
				len(buf), err, matcher.ErrDumpTooLarge)
		}
		if want := []int{16, 32, 64, 100}; !slices.Equal(*given, want) {
			t.Errorf("dumped into buffers of %v bytes, want %v", *given, want)
		}
	})
}
