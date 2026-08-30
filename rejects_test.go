// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestRejects(t *testing.T) {
	t.Parallel()

	t.Run("Rejects", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the driven check's own message", func(t *testing.T) {
			t.Parallel()

			got := assert.Rejects(t, "rejects 2", func(tb assert.TB) {
				assert.Equal(tb, 2, 1, "the value is one")
			})

			if !strings.Contains(got, "the value is one") {
				t.Fatalf("returned %q, want the check's own message", got)
			}
		})

		t.Run("reports when the driven check passes", func(t *testing.T) {
			t.Parallel()

			outer := &matchertest.Seat{}
			assert.Rejects(outer, "rejects 1", func(tb assert.TB) {
				assert.Equal(tb, 1, 1, "the value is one")
			})

			if !outer.Failed() {
				t.Fatal("reported nothing when the driven check passed")
			}
			if !strings.Contains(outer.First(), "must reject") {
				t.Fatalf("failure %q does not name the rejection that did not happen", outer.First())
			}
		})

		t.Run("stops the body at its first failure", func(t *testing.T) {
			t.Parallel()

			reached := false
			assert.Rejects(t, "stops at the first failure", func(tb assert.TB) {
				assert.Equal(tb, 2, 1, "the value is one")
				reached = true
			})

			if reached {
				t.Fatal("the body ran past an assertion it had already failed")
			}
		})

		t.Run("leaves no goroutine behind", func(t *testing.T) {
			t.Parallel()

			// Rejects drives its body on a goroutine so Goexit has one
			// to end. Returning before that goroutine finishes would
			// leak it into every test that follows.
			done := make(chan struct{})
			go func() {
				defer close(done)
				assert.Rejects(t, "rejects 2", func(tb assert.TB) {
					assert.Equal(tb, 2, 1, "the value is one")
				})
			}()
			<-done
		})
	})
}
