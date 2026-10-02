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

		t.Run("reports a record of rejects when the driven check passes", func(t *testing.T) {
			t.Parallel()

			outer := &matchertest.Seat{}
			assert.Rejects(outer, "rejects 1", func(tb assert.TB) {
				assert.Equal(tb, 1, 1, "the value is one")
			})

			if len(outer.Fatals()) != 1 || len(outer.Errs()) != 0 {
				t.Fatalf("reported %q through Fatalf and %q through Errorf, want one failure that stops the test",
					outer.Fatals(), outer.Errs())
			}
			records := outer.Records()
			if len(records) != 1 || records[0].Assertion != "rejects" || records[0].Contract != "rejects 1" {
				t.Fatalf("reported %+v, want one record of rejects whose contract is the message", records)
			}
			if len(records[0].Detail) != 0 {
				t.Fatalf("the record states %v, and rejects declares no detail field", records[0].Detail)
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
