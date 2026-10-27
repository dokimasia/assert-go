// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/prop"
)

// skipping is a seat that keeps the reason of each skip, as a test of the
// package testing logs it, and takes each failure as its record.
type skipping struct {
	matchertest.Seat

	// mu guards skips.
	mu sync.Mutex
	// skips are the reasons of the skips, in call order.
	skips []string
}

// Skipf keeps the formatted reason, and returns.
func (s *skipping) Skipf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skips = append(s.skips, fmt.Sprintf(format, args...))
}

// reasons returns a copy of the reasons of the skips.
func (s *skipping) reasons() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.skips)
}

// waitsInBubble is a body that waits for a goroutine with synctest.Wait, and
// fails for each drawn x. Outside a testing/synctest bubble, synctest.Wait
// panics.
func waitsInBubble(c *prop.Case) {
	x := c.Draw(prop.Integer(2, 3), "x")
	done := make(chan int, 1)
	go func() { done <- x }()
	synctest.Wait()
	assert.Equal(c, <-done, 1, "x is 1")
}

// recordingTest returns a wrapper of Around that runs each run in a bubble,
// and keeps in wrapped the test that it receives. The wrapper of each run
// receives the same test.
func recordingTest(wrapped **testing.T) func(*testing.T, func(*testing.T)) {
	var mu sync.Mutex
	return func(t *testing.T, run func(*testing.T)) {
		t.Helper()
		mu.Lock()
		*wrapped = t
		mu.Unlock()
		synctest.Test(t, run)
	}
}

// TestProperty checks how a call of ForAll delivers the faults of its store
// and its record to its seat, and what it stores.
func TestProperty(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("runs each case under Around in a bubble on the seat of the check of Rejects", func(t *testing.T) {
			t.Parallel()
			var wrapped *testing.T
			failures := assert.Rejects(t, "the property fails", func(seat assert.TB) {
				prop.ForAll(seat, contract, waitsInBubble, prop.Seed(7), prop.Around(recordingTest(&wrapped)))
			})
			assert.Length(t, failures, 1, "the record of the property")
			assert.Equal(t, failures[0].Assertion, "prop-for-all", "the property fails")
			assert.True(t, wrapped == t, "the wrapper receives the test of Rejects")
		})

		t.Run("runs each case under Around in a bubble on a recorder of a test", func(t *testing.T) {
			t.Parallel()
			var wrapped *testing.T
			rec := assert.NewRecorder().WithTest(t)
			prop.ForAll(rec, contract, waitsInBubble, prop.Seed(7), prop.Around(recordingTest(&wrapped)))
			assert.Empty(t, rec.Faults(), "no fault")
			assert.Length(t, rec.Failures(), 1, "the record of the property")
			assert.True(t, wrapped == t, "the wrapper receives the test of the recorder")
		})

		t.Run("passes the fault of each skipped file to the seat in the order of the files", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "b.json"), `{"store": 2}`)
			write(t, filepath.Join(dir, "a.json"), `{"store": 3}`)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Seed(7), prop.Store(dir))
			faults := seat.Faults()
			assert.Length(t, faults, 2, "a fault for each file")
			expectFault(t, faults[0], laterFault(forAllOp, dir, "a.json", "3"))
			expectFault(t, faults[1], laterFault(forAllOp, dir, "b.json", "2"))
		})

		t.Run("drops the notes of a seat without a log", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, draws(prop.Integer(0, 9)), prop.Seed(7), prop.Store(dir))
			assert.False(t, rec.Failed(), "the run passes without a note")
		})

		t.Run("reports a failing run and the fault of each skipped file", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, failsFrom(1001), prop.Seed(7), prop.Store(dir))
			expectOnlyFault(t, seat.Faults(), laterFault(forAllOp, dir, "later.json", "2"))
			assert.Length(t, seat.Records(), 1, "the record")
		})

		t.Run("skips the first seat of the chain with Skipf with the reason of the run", func(t *testing.T) {
			t.Parallel()
			seat := &skipping{}
			prop.ForAll(wrapper{seat}, contract, func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), drawn)
				c.Skipf("the file system refuses direct I/O")
			}, prop.Seed(7))
			assert.Equal(t, seat.reasons(), []string{"the file system refuses direct I/O"}, "the reason of the skip")
			assert.False(t, seat.Failed(), "no failure")
		})

		t.Run("skips a test with the reason of its run", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				prop.ForAll(t, contract, func(c *prop.Case) {
					c.Draw(prop.Integer(0, 9), drawn)
					c.Skipf("the file system refuses direct I/O")
				}, prop.Seed(7), prop.Store(t.TempDir()))
				return
			}
			out, err := childtest.Run(t, t.Name())
			assert.NoError(t, err, "the child exits with status 0:\n"+out)
			assert.Contains(t, out, "--- SKIP: "+t.Name()+" ", "the child skips the test")
			assert.Contains(t, out, "the file system refuses direct I/O", "the child logs the reason")
		})

		t.Run("stores no counterexample in a test binary that a mutation run instrumented", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				dir := t.TempDir()
				got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Store(dir))
				assert.Equal(t, got[choicesField], any("prop1:AOkH"), "the counterexample of seed 7")
				assert.Empty(t, loaded(t, dir).Entries, "no entry of the counterexample")
				return
			}
			inMutationRun(t)
		})
	})
}
