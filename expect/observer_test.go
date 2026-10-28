// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"reflect"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

// watcher is a seat that wraps a recorder and observes each assertion call.
type watcher struct {
	expect.TB

	// mu guards calls.
	mu sync.Mutex
	// calls are the observed calls, in order.
	calls []expect.Call
}

// ObserveCall keeps call.
func (w *watcher) ObserveCall(call expect.Call) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, call)
}

// Unwrap returns the wrapped seat.
func (w *watcher) Unwrap() assert.TB { return w.TB }

// TestObserver checks the observer of the recording surface.
func TestObserver(t *testing.T) {
	t.Parallel()

	t.Run("Observer", func(t *testing.T) {
		t.Parallel()

		t.Run("is the observer of the aborting surface", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.Observer](), reflect.TypeFor[assert.Observer](), "one observer")
		})

		t.Run("receives a failing call of this surface", func(t *testing.T) {
			t.Parallel()
			w := &watcher{TB: expect.NewRecorder()}
			expect.Equal(w, 1, 2, "the totals match")
			expect.Equal(t, w.calls, []expect.Call{{Assertion: "equal", Contract: "the totals match"}}, "the call")
		})
	})

	t.Run("Call", func(t *testing.T) {
		t.Parallel()

		t.Run("is the call of the aborting surface", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.Call](), reflect.TypeFor[assert.Call](), "one call")
		})
	})
}
