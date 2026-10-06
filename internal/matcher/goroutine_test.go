// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"math"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The stacks that outgrow the largest buffer of a leak check: deep
// goroutines whose frames each print ten words of arguments. They take more
// than 64 MiB with or without the race detector, measured at 19 KiB and
// 31 KiB a goroutine, and they are fewer than the 8,128 goroutines that the
// race detector allows.
const (
	// deepGoroutines is the number of deep goroutines.
	deepGoroutines = 5000
	// deepFrames is the depth of each, of which a dump prints 100 frames.
	deepFrames = 120
)

// leakMsg is the caller's message of the leak check over deep stacks.
const leakMsg = "the worker stops"

// The deep goroutines that every reading of a check finds, and the most that
// the check of their leak allocates. Their stacks take 1.3 MiB, and 2.2 MiB
// under the race detector, so their dump outgrows the first buffer of
// 1 MiB, and the check grows one buffer to 2 or 4 MiB.
const (
	leakedGoroutines = 80
	maxCheckBytes    = 32 << 20
)

// TestGoroutine reads the goroutines of the whole process, so neither it
// nor its cases run in parallel. See [matchertest.RunNoGoroutineLeaks].
func TestGoroutine(t *testing.T) {
	t.Run("NoGoroutineLeaks", func(t *testing.T) {
		matchertest.RunNoGoroutineLeaks(t, func(s *matchertest.Seat, msg string) func() {
			return matcher.NoGoroutineLeaks(s, matcher.Fatal, msg)
		})

		t.Run("writes the record of a passing check", func(t *testing.T) {
			checkPassRecord(t, "no-task-leaks", func(seat matcher.Seat) {
				matcher.NoGoroutineLeaks(seat, matcher.Fatal, leakMsg)()
			})
		})
	})

	t.Run("NoGoroutineLeaks ends without a verdict on stacks that outgrow 64 MiB", func(t *testing.T) {
		seat := newKeepingSeat()
		check := matcher.NoGoroutineLeaks(seat, matcher.Fatal, leakMsg)
		end := runDeep(t, deepGoroutines)
		check()
		end()

		checkStacksFault(t, seat)
	})

	t.Run("NoGoroutineLeaks ends without a verdict on stacks that outgrow 64 MiB at its call", func(t *testing.T) {
		seat := newKeepingSeat()
		end := runDeep(t, deepGoroutines)
		check := matcher.NoGoroutineLeaks(seat, matcher.Fatal, leakMsg)
		end()
		check()

		checkStacksFault(t, seat)
	})

	// A check that read each dump into a buffer of its own would allocate
	// 2 MiB at each of its 100 readings.
	t.Run("NoGoroutineLeaks reads the dumps of a check into one buffer", func(t *testing.T) {
		seat := &matchertest.Seat{}
		check := matcher.NoGoroutineLeaks(seat, matcher.Soft, leakMsg)
		end := runDeep(t, leakedGoroutines)
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		check()
		runtime.ReadMemStats(&after)
		end()

		if records := seat.Records(); len(records) != 1 || records[0].Assertion != "no-task-leaks" {
			t.Fatalf("reported %+v, want the leak of the deep goroutines", records)
		}
		if grown := after.TotalAlloc - before.TotalAlloc; grown > maxCheckBytes {
			t.Errorf("the check allocated %d bytes, want at most %d", grown, maxCheckBytes)
		}
	})
}

// checkStacksFault checks that seat received the fault of a check whose
// stacks do not fit, and the call record of the error alone.
func checkStacksFault(t *testing.T, seat *keepingSeat) {
	t.Helper()
	if fatals := seat.Fatals(); len(fatals) != 1 {
		t.Fatalf("reported %q, want one report that the stacks do not fit", fatals)
	}
	if records := seat.Records(); len(records) != 0 {
		t.Errorf("reported the records %+v for a check that states no verdict", records)
	}
	lines := seat.lines(t)
	if len(lines) != 1 || lines[0]["verdict"] != "error" || lines[0]["contract"] != leakMsg ||
		!strings.Contains(lines[0]["error"].(string), "the stacks of every goroutine fill 67108864 bytes") {
		t.Errorf("wrote the call records %v, want one error of the check", lines)
	}
}

// runDeep starts n deep goroutines, waits until each is at the bottom of its
// stack, and returns the function that ends them and waits for them.
func runDeep(t *testing.T, n int) (end func()) {
	t.Helper()
	var ready, ended sync.WaitGroup
	release := make(chan struct{})
	ready.Add(n)
	for range n {
		ended.Go(func() {
			m := uint64(math.MaxUint64)
			deepStack(m, m, m, m, m, m, m, m, m, m, deepFrames, &ready, release)
		})
	}
	ready.Wait()
	return func() {
		close(release)
		ended.Wait()
	}
}

// deepStack recurses frames deep, each frame with ten words of arguments
// that a dump prints, marks ready at the bottom, and waits for release.
//
//go:noinline
func deepStack(
	//nolint:unparam // the ten words lengthen each frame that a dump prints
	a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 uint64,
	frames int, ready *sync.WaitGroup, release <-chan struct{},
) {
	if frames == 0 {
		ready.Done()
		<-release
		return
	}
	deepStack(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, frames-1, ready, release)
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

// TestDumpAll checks the buffers that a dump of every stack is read into.
func TestDumpAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		giveFirst   int
		giveLimit   int
		giveSize    int
		wantBuffers []int
	}{
		{
			name:      "dumps once into a buffer that the dump does not fill",
			giveFirst: 16, giveLimit: 1024, giveSize: 3, wantBuffers: []int{16},
		},
		{
			name:      "doubles the buffer until a dump leaves room",
			giveFirst: 16, giveLimit: 1024, giveSize: 100, wantBuffers: []int{16, 32, 64, 128},
		},
		{
			name:      "dumps again after a dump that fills its buffer exactly",
			giveFirst: 16, giveLimit: 1024, giveSize: 16, wantBuffers: []int{16, 32},
		},
		{
			name:      "caps the last buffer at the limit",
			giveFirst: 16, giveLimit: 100, giveSize: 99, wantBuffers: []int{16, 32, 64, 100},
		},
		{
			name:      "grows an empty buffer",
			giveFirst: 0, giveLimit: 1024, giveSize: 3, wantBuffers: []int{0, 1, 2, 4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dump, given := sized(tt.giveSize)
			buf, n, err := matcher.DumpAll(make([]byte, tt.giveFirst), tt.giveLimit, dump)
			if err != nil {
				t.Fatalf("DumpAll() = %v, want the whole dump", err)
			}
			if n != tt.giveSize || len(buf) != tt.wantBuffers[len(tt.wantBuffers)-1] {
				t.Errorf("DumpAll() = %d bytes in a buffer of %d, want %d in the last buffer",
					n, len(buf), tt.giveSize)
			}
			if !slices.Equal(*given, tt.wantBuffers) {
				t.Errorf("dumped into buffers of %v bytes, want %v", *given, tt.wantBuffers)
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

// dumped keeps the buffer that a call of DumpAll returns.
var dumped []byte

// TestGoroutineAllocs checks the allocation ceiling of a passing call of
// each function of goroutine.go. A leak check allocates more for more
// goroutines, so the test checks the ceilings in a child process, which
// runs no other test.
func TestGoroutineAllocs(t *testing.T) {
	if !childtest.InChild(t) {
		runChild(t)
		return
	}
	checkAllocs(t, goroutineCases())
}

// BenchmarkGoroutine measures a passing call of each function of
// goroutine.go.
func BenchmarkGoroutine(b *testing.B) {
	benchAllocs(b, goroutineCases())
}

// goroutineCases returns a passing call of each function of goroutine.go,
// with its allocation ceiling, measured: a leak check that finds no new
// goroutine, and a dump of three bytes into a buffer that it does not fill.
func goroutineCases() []allocCase {
	buf := make([]byte, 64)
	dump := func(b []byte) int { return copy(b, "abc") }
	return []allocCase{
		{name: "NoGoroutineLeaks", allocs: 7, call: func(seat matcher.Seat) {
			matcher.NoGoroutineLeaks(seat, matcher.Fatal, leakMsg)()
		}},
		{name: "DumpAll", call: func(matcher.Seat) { dumped, _, _ = matcher.DumpAll(buf, 1024, dump) }},
	}
}
