// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
)

func TestRecorder(t *testing.T) {
	t.Parallel()

	t.Run("Fatalf", func(t *testing.T) {
		t.Parallel()

		t.Run("records the message instead of aborting", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("boom %d", 7)

			if !r.Failed() {
				t.Fatal("Failed() = false, want true")
			}
			if got, want := r.Message(), "boom 7"; got != want {
				t.Fatalf("Message() = %q, want %q", got, want)
			}
		})

		t.Run("keeps the first message", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("first")
			r.Fatalf("second")

			if got, want := r.Message(), "first"; got != want {
				t.Fatalf("Message() = %q, want %q", got, want)
			}
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		t.Parallel()

		t.Run("records every message", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Errorf("one")
			r.Errorf("two")

			if got, want := len(r.Messages()), 2; got != want {
				t.Fatalf("len(Errors()) = %d, want %d", got, want)
			}
		})

		t.Run("marks the recorder failed", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Errorf("one")

			if !r.Failed() {
				t.Fatal("Failed() = false, want true")
			}
		})
	})

	t.Run("Failed", func(t *testing.T) {
		t.Parallel()

		t.Run("false before anything is recorded", func(t *testing.T) {
			t.Parallel()

			if assert.NewRecorder().Failed() {
				t.Fatal("Failed() = true on a fresh recorder")
			}
		})
	})

	t.Run("HelperCalls", func(t *testing.T) {
		t.Parallel()

		t.Run("counts each call", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Helper()
			r.Helper()

			if got, want := r.HelperCalls(), 2; got != want {
				t.Fatalf("HelperCalls() = %d, want %d", got, want)
			}
		})
	})
}

// A Recorder stands in for *testing.T so the example prints the
// outcome instead of failing.
func ExampleNewRecorder() {
	r := assert.NewRecorder()
	fmt.Println(r.Failed())
	// Output: false
}

func TestRecorderClock(t *testing.T) {
	t.Parallel()

	t.Run("Clock", func(t *testing.T) {
		t.Parallel()

		t.Run("answers the runtime clock by default", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			if got := r.Clock().Now(); got.Before(epoch) {
				t.Fatalf("Now() = %v, want a runtime reading", got)
			}
		})

		t.Run("answers what WithClock set", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder().WithClock(assert.NewControlled(epoch))
			if got := r.Clock().Now(); !got.Equal(epoch) {
				t.Fatalf("Now() = %v, want the controlled clock's %v", got, epoch)
			}
		})
	})

	t.Run("Failures", func(t *testing.T) {
		t.Parallel()

		t.Run("answers every record in call order", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Equal(r, 1, 2, "the values match")

			records := r.Failures()
			if len(records) != 1 {
				t.Fatalf("Failures() holds %d records, want 1", len(records))
			}
			if got, want := records[0].Assertion, "equal"; got != want {
				t.Fatalf("Assertion = %q, want %q", got, want)
			}
		})

		t.Run("holds nothing for a message reported without a record", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("a bare message")

			if got := len(r.Failures()); got != 0 {
				t.Fatalf("Failures() holds %d records, want none", got)
			}
		})
	})
}
