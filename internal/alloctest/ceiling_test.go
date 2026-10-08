// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package alloctest_test

import (
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matcher"
)

// iterations is the number of iterations of a fake benchmark: enough that
// the allocations the runtime makes for itself during a run round to none
// per iteration.
const iterations = 1000

// failingAllocs is the ceiling of a call that fails: a failure allocates its
// record and its text, and the ceiling leaves room for both, so the case
// checks the failure alone.
const failingAllocs = 1_000_000

// sink keeps the allocation of a call alive, so the compiler moves it to
// the heap.
var sink []byte

// fakeB is a benchmark of a fixed number of iterations, which records what
// a contract reports to it.
type fakeB struct {
	*assert.Recorder
	// remaining is the number of iterations left.
	remaining int
}

// Loop reports whether another iteration runs, and counts it.
func (b *fakeB) Loop() bool {
	if b.remaining == 0 {
		return false
	}
	b.remaining--
	return true
}

// ReportMetric discards the metric.
func (*fakeB) ReportMetric(float64, string) {}

// TestCeilingAllocs checks a table of ceilings in a test and in a benchmark. It
// does not run in parallel: expect.MaxAllocs and a contract count the
// allocations of the whole process.
func TestCeilingAllocs(t *testing.T) {
	passing := alloctest.Case{Name: "noop", Call: func(assert.TB) {}}
	allocating := alloctest.Case{Name: "allocating", Call: func(assert.TB) { sink = make([]byte, 64) }}
	failing := alloctest.Case{
		Name:   "failing",
		Call:   func(tb assert.TB) { assert.True(tb, false, "the flag is set") },
		Allocs: failingAllocs,
	}

	t.Run("Check", func(t *testing.T) {
		t.Run("passes a call within its ceiling", func(t *testing.T) {
			rec := assert.NewRecorder()
			alloctest.Check(rec, []alloctest.Case{passing})
			assert.False(t, rec.Failed(), "a call that allocates nothing meets a ceiling of 0")
		})

		t.Run("fails a call past its ceiling in a build that counts allocations", func(t *testing.T) {
			rec := assert.NewRecorder()
			alloctest.Check(rec, []alloctest.Case{allocating})
			assert.Equal(t, assertionsOf(rec.Failures()), assertionsIn(matcher.AllocationsCounted(), "max-allocs"),
				"the failure of the ceiling, in a build that checks it")
		})

		t.Run("fails a call that fails", func(t *testing.T) {
			rec := assert.NewRecorder()
			alloctest.Check(rec, []alloctest.Case{failing})
			failures := rec.Failures()
			assert.Length(t, failures, 1, "one failure")
			assert.Equal(t, []string{failures[0].Assertion, failures[0].Contract}, []string{"false", "failing passes"},
				"the failure names the case")
		})
	})

	t.Run("Measure", func(t *testing.T) {
		t.Run("calls the case once before the iterations and in each iteration, and passes a call within its ceiling",
			func(t *testing.T) {
				var calls int
				counted := alloctest.Case{Name: "counted", Call: func(assert.TB) { calls++ }}
				b := &fakeB{Recorder: assert.NewRecorder(), remaining: iterations}
				alloctest.Measure(b, counted)
				assert.Equal(t, calls, iterations+1, "one call before the iterations and one in each")
				assert.False(t, b.Failed(), "a call that allocates nothing meets a ceiling of 0")
			})

		t.Run("counts no allocation of the setup of a first call", func(t *testing.T) {
			var once sync.Once
			lazy := alloctest.Case{Name: "lazy", Call: func(assert.TB) {
				once.Do(func() {
					for range 2 * iterations {
						sink = make([]byte, 64)
					}
				})
			}}
			b := &fakeB{Recorder: assert.NewRecorder(), remaining: iterations}
			alloctest.Measure(b, lazy)
			assert.False(t, b.Failed(), "the first call's two allocations per iteration fall before the contract")
		})

		t.Run("fails a call past its ceiling in a build that counts allocations", func(t *testing.T) {
			b := &fakeB{Recorder: assert.NewRecorder(), remaining: iterations}
			alloctest.Measure(b, allocating)
			assert.Equal(t, assertionsOf(b.Failures()), assertionsIn(matcher.AllocationsCounted(), "bench-max-allocs"),
				"the failure of the ceiling, in a build that checks it")
		})

		t.Run("fails a call that fails", func(t *testing.T) {
			b := &fakeB{Recorder: assert.NewRecorder(), remaining: iterations}
			alloctest.Measure(b, failing)
			failures := b.Failures()
			assert.Length(t, failures, 1, "one failure")
			assert.Equal(t, []string{failures[0].Assertion, failures[0].Contract}, []string{"false", "failing passes"},
				"the failure names the case")
		})
	})
}

// assertionsOf returns the assertion of each failure, in order.
func assertionsOf(failures []assert.Failure) []string {
	out := make([]string, len(failures))
	for i, f := range failures {
		out[i] = f.Assertion
	}
	return out
}

// assertionsIn returns the assertion id alone when counted is true, and no
// assertion otherwise.
func assertionsIn(counted bool, id string) []string {
	if counted {
		return []string{id}
	}
	return []string{}
}
