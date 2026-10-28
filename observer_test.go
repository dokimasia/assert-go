// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

// watcher is the handle of a generated check: it wraps the seat of its test
// and observes each assertion call. It also keeps, in order, the observed
// calls and the failures that reach the seat it wraps.
type watcher struct {
	assert.TB

	// mu guards events.
	mu sync.Mutex
	// calls are the observed calls, in order.
	calls []assert.Call
	// events are "observe" for each observed call and "report" for each
	// failure that the wrapped seat receives, in order.
	events []string
}

// ObserveCall keeps call.
func (w *watcher) ObserveCall(call assert.Call) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, call)
	w.events = append(w.events, "observe")
}

// Unwrap returns the wrapped seat.
func (w *watcher) Unwrap() assert.TB { return w.TB }

// contracts returns the contracts of the observed calls, in order.
func (w *watcher) contracts() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, len(w.calls))
	for i, c := range w.calls {
		out[i] = c.Contract
	}
	return out
}

// reporting is a seat that keeps each failure that it receives in the
// events of its watcher.
type reporting struct {
	*assert.Recorder

	w *watcher
}

// Report notes the failure in the watcher's events, and keeps it.
func (r reporting) Report(f assert.Failure, aborting bool) {
	r.w.mu.Lock()
	r.w.events = append(r.w.events, "report")
	r.w.mu.Unlock()
	r.Recorder.Report(f, aborting)
}

// TestObserver checks which assertion calls the observer of a seat's chain
// receives.
func TestObserver(t *testing.T) {
	t.Parallel()

	t.Run("Observer", func(t *testing.T) {
		t.Parallel()

		t.Run("receives a passing call and a failing call with their assertion and contract", func(t *testing.T) {
			t.Parallel()
			w := &watcher{TB: assert.NewRecorder()}
			assert.Equal(w, 1, 1, "the counts match")
			assert.Equal(w, 1, 2, "the totals match")
			assert.Equal(t, w.calls, []assert.Call{
				{Assertion: "equal", Contract: "the counts match"},
				{Assertion: "equal", Contract: "the totals match"},
			}, "both calls")
		})

		t.Run("receives a failing call before the seat receives its failure", func(t *testing.T) {
			t.Parallel()
			w := &watcher{}
			w.TB = reporting{Recorder: assert.NewRecorder(), w: w}
			assert.True(w, false, "the flag is set")
			assert.Equal(t, w.events, []string{"observe", "report"}, "the observation first")
		})

		t.Run("receives a call that ends with a fault", func(t *testing.T) {
			t.Parallel()
			w := &watcher{TB: assert.NewRecorder()}
			_ = assert.ErrorAs[int](w, errMissingOption, "the error is an int")
			assert.Equal(t, w.calls, []assert.Call{{Assertion: "err-as", Contract: "the error is an int"}},
				"the call of the fault")
		})

		t.Run("does not receive a call of Fault, which is no assertion", func(t *testing.T) {
			t.Parallel()
			w := &watcher{TB: assert.NewRecorder()}
			assert.Fault(w, errMissingOption)
			assert.Empty(t, w.calls, "no call")
		})

		t.Run("receives the calls in the cases of a property, then the property", func(t *testing.T) {
			t.Parallel()
			w := &watcher{TB: assert.NewRecorder()}
			prop.ForAll(w, "every value is small", func(c *prop.Case) {
				assert.True(c, c.Draw(prop.Integer(0, 9), "x") < 10, "the value is small")
			}, prop.Seed(7), prop.Cases(3))
			contracts := w.contracts()
			assert.Equal(t, contracts[len(contracts)-1], "every value is small", "the property last")
			assert.Contains(t, contracts, "the value is small", "a call in a case")
		})

		t.Run("receives a property that skips", func(t *testing.T) {
			t.Parallel()
			w := &watcher{TB: assert.NewRecorder()}
			prop.ForAll(w, "the file system supports direct I/O", func(c *prop.Case) {
				c.Draw(prop.Integer(0, 9), "x")
				c.Skipf("the file system refuses direct I/O")
			}, prop.Seed(7))
			assert.Equal(t, w.contracts(), []string{"the file system supports direct I/O"}, "the property")
		})

		t.Run("receives the calls in the check of Rejects, then Rejects", func(t *testing.T) {
			t.Parallel()
			w := &watcher{TB: assert.NewRecorder()}
			assert.Rejects(w, "the check rejects the subject", func(s assert.TB) {
				assert.True(s, false, "the subject is sound")
			})
			assert.Equal(t, w.contracts(), []string{"the subject is sound", "the check rejects the subject"},
				"the check's call, then Rejects")
		})
	})
}
