// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"context"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// TestSubject checks the shapes of a built behaviour that no corpus case
// calls. The corpus cases call every shape that an assertion of the
// definition takes.
func TestSubject(t *testing.T) {
	t.Parallel()

	t.Run("Subjects", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a raises whose context shape panics", func(t *testing.T) {
			t.Parallel()
			defer func() {
				if recover() == nil {
					t.Fatal("the context shape of raises returns, want a panic")
				}
			}()
			_ = conformance.Subjects["raises"]().Ctx(context.Background())
		})

		t.Run("returns a settles-after that fails twice before it passes", func(t *testing.T) {
			t.Parallel()
			seated := conformance.Subjects["settles-after"]().Seated
			for attempt, want := range []bool{true, true, false} {
				r := assert.NewRecorder()
				seated(r)
				if r.Failed() != want {
					t.Fatalf("attempt %d fails %t, want %t", attempt+1, r.Failed(), want)
				}
			}
		})
	})
}
