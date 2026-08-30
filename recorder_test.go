// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
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
			if got, want := r.Msg(), "boom 7"; got != want {
				t.Fatalf("Msg() = %q, want %q", got, want)
			}
		})

		t.Run("keeps the first message", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			r.Fatalf("first")
			r.Fatalf("second")

			if got, want := r.Msg(), "first"; got != want {
				t.Fatalf("Msg() = %q, want %q", got, want)
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

			if got, want := len(r.Errors()), 2; got != want {
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
