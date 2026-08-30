// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// Compile-time proof that the public seat satisfies the internal one.
// The two are declared separately so neither package depends on the
// other for its interface.
var _ matcher.Seat = assert.TB(nil)

func TestDoc(t *testing.T) {
	t.Parallel()

	t.Run("satisfaction", func(t *testing.T) {
		t.Parallel()

		t.Run("a Recorder drives a matcher", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			matcher.Report(r, matcher.Fatal, "reported")

			if got, want := r.Msg(), "reported"; got != want {
				t.Fatalf("Msg() = %q, want %q", got, want)
			}
		})
	})
}
