// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"runtime"
	"slices"
	"testing"
	"weak"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// cleaningSeat is a seat that keeps the functions that its Cleanup registers,
// and records what was reported.
type cleaningSeat struct {
	matchertest.Seat

	cleanups matcher.Cleanups
}

// Cleanup registers f with the seat's cleanups.
func (s *cleaningSeat) Cleanup(f func()) { s.cleanups.Add(f) }

func TestCleanup(t *testing.T) {
	t.Parallel()

	t.Run("Cleanup", func(t *testing.T) {
		t.Parallel()

		t.Run("registers f with a seat that has Cleanup", func(t *testing.T) {
			t.Parallel()

			var tr trace
			seat := &cleaningSeat{}
			matcher.Cleanup(seat, "assert.Cleanup", tr.step("released"))
			seat.cleanups.Run()
			if got := tr.ran(); !slices.Equal(got, []string{"released"}) {
				t.Fatalf("the seat ran %v, want the function that Cleanup registered", got)
			}
		})

		t.Run("registers f with the first seat of the chain that has Cleanup", func(t *testing.T) {
			t.Parallel()

			var tr trace
			seat := &cleaningSeat{}
			matcher.Cleanup(wrapper{wrapper{seat}}, "assert.Cleanup", tr.step("released"))
			seat.cleanups.Run()
			if got := tr.ran(); !slices.Equal(got, []string{"released"}) {
				t.Fatalf("the wrapped seat ran %v, want the function that Cleanup registered", got)
			}
		})

		t.Run("ends the call with a fault of op on a chain without Cleanup", func(t *testing.T) {
			t.Parallel()

			var tr trace
			seat := &matchertest.Seat{}
			matcher.Cleanup(wrapper{seat}, "assert.Cleanup", tr.step("released"))
			faults := seat.Faults()
			var got *fault.Error
			if len(faults) != 1 || !errors.As(faults[0], &got) || got.Op != "assert.Cleanup" ||
				got.Reason != "the seat has no Cleanup" {

				t.Fatalf("the seat received the faults %v, want the fault of assert.Cleanup", faults)
			}
			if steps := tr.ran(); len(steps) != 0 {
				t.Fatalf("the call ran %v, want no function", steps)
			}
		})
	})
}

func TestCleanups(t *testing.T) {
	t.Parallel()

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps f until a run", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("released"))
			if got := tr.ran(); len(got) != 0 {
				t.Fatalf("Add ran %v, want no function before a run", got)
			}
		})

		t.Run("runs f at once after Close", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("closed"))
			c.Close()
			c.Add(tr.step("late"))
			if got := tr.ran(); !slices.Equal(got, []string{"closed", "late"}) {
				t.Fatalf("Add after Close ran %v, want the function at once", got)
			}
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the functions, the last registered first", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("first"))
			c.Add(tr.step("second"))
			c.Run()
			if got := tr.ran(); !slices.Equal(got, []string{"second", "first"}) {
				t.Fatalf("Run ran %v, want the last registered first", got)
			}
		})

		t.Run("forgets the functions that it ran", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("once"))
			c.Run()
			c.Run()
			if got := tr.ran(); !slices.Equal(got, []string{"once"}) {
				t.Fatalf("two runs ran %v, want the function once", got)
			}
		})

		t.Run("runs a function that a function registers in the same pass", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("first"))
			c.Add(func() {
				c.Add(tr.step("registered by second"))
				tr.step("second")()
			})
			c.Run()
			want := []string{"second", "registered by second", "first"}
			if got := tr.ran(); !slices.Equal(got, want) {
				t.Fatalf("Run ran %v, want %v", got, want)
			}
		})

		t.Run("leaves a function that arrives after the run to the next run", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("first run"))
			c.Run()
			c.Add(tr.step("late"))
			if got := tr.ran(); !slices.Equal(got, []string{"first run"}) {
				t.Fatalf("Add after Run ran %v, want no function before the next run", got)
			}
			c.Run()
			if got := tr.ran(); !slices.Equal(got, []string{"first run", "late"}) {
				t.Fatalf("the next run ran %v, want the late function", got)
			}
		})

		t.Run("leaves a closed list closed", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Close()
			c.Run()
			c.Add(tr.step("late"))
			if got := tr.ran(); !slices.Equal(got, []string{"late"}) {
				t.Fatalf("Add after Close and Run ran %v, want the function at once", got)
			}
		})

		t.Run("releases each function that it ran", func(t *testing.T) {
			t.Parallel()

			var c matcher.Cleanups
			held := func() weak.Pointer[[64]int] {
				value := new([64]int)
				c.Add(func() { value[0]++ })
				return weak.Make(value)
			}()
			c.Run()
			runtime.GC()
			if held.Value() != nil {
				t.Fatal("the list keeps the value of a function that it ran, want it released")
			}
			runtime.KeepAlive(&c)
		})

		t.Run("runs the earlier functions when a function ends its goroutine", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("first"))
			c.Add(func() {
				tr.step("second")()
				runtime.Goexit()
			})
			done := make(chan struct{})
			go func() {
				defer close(done)
				c.Run()
			}()
			<-done
			if got := tr.ran(); !slices.Equal(got, []string{"second", "first"}) {
				t.Fatalf("Run ran %v, want every function despite the end of the goroutine", got)
			}
		})

		t.Run("runs the earlier functions when a function panics, and lets the panic go on", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("first"))
			c.Add(func() { panic("the release failed") })
			recovered := func() (r any) {
				defer func() { r = recover() }()
				c.Run()
				return nil
			}()
			if got := tr.ran(); recovered != "the release failed" || !slices.Equal(got, []string{"first"}) {
				t.Fatalf("Run ran %v and panicked with %v, want the first function and the panic", got, recovered)
			}
		})
	})

	t.Run("Close", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the functions, the last registered first", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(tr.step("first"))
			c.Add(tr.step("second"))
			c.Close()
			if got := tr.ran(); !slices.Equal(got, []string{"second", "first"}) {
				t.Fatalf("Close ran %v, want the last registered first", got)
			}
		})

		t.Run("runs a function that a function registers before the list closes", func(t *testing.T) {
			t.Parallel()

			var tr trace
			var c matcher.Cleanups
			c.Add(func() {
				c.Add(tr.step("registered"))
				tr.step("registering")()
			})
			c.Close()
			if got := tr.ran(); !slices.Equal(got, []string{"registering", "registered"}) {
				t.Fatalf("Close ran %v, want the registered function after the one that registered it", got)
			}
		})
	})
}

// TestCleanupAllocs checks the allocation ceiling of each function of
// cleanup.go.
func TestCleanupAllocs(t *testing.T) {
	checkAllocs(t, cleanupCases())
}

// BenchmarkCleanup measures each function of cleanup.go.
func BenchmarkCleanup(b *testing.B) {
	benchAllocs(b, cleanupCases())
}

// noCleanup is a function that a cleanup of the allocation cases runs.
func noCleanup() {}

// cleanupCases returns a call of each function of cleanup.go, with its
// allocation ceiling. A list keeps the capacity that its first run left, so a
// registration that a run follows allocates nothing after the first call.
func cleanupCases() []allocCase {
	seat := &cleaningSeat{}
	var listed, closed matcher.Cleanups
	closed.Close()
	return []allocCase{
		{name: "Cleanup", call: func(matcher.Seat) {
			matcher.Cleanup(seat, "assert.Cleanup", noCleanup)
			seat.cleanups.Run()
		}},
		{name: "Cleanups.Add", call: func(matcher.Seat) { closed.Add(noCleanup) }},
		{name: "Cleanups.Run", call: func(matcher.Seat) {
			listed.Add(noCleanup)
			listed.Run()
		}},
		{name: "Cleanups.Close", call: func(matcher.Seat) { closed.Close() }},
	}
}
