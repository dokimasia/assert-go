// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// The public seat satisfies this one by method set alone, which is
// what lets neither package import the other for its interface.
var _ matcher.Seat = assert.TB(nil)

func TestSeat(t *testing.T) {
	t.Parallel()

	t.Run("Mode", func(t *testing.T) {
		t.Parallel()

		t.Run("the zero value is Fatal", func(t *testing.T) {
			t.Parallel()

			var zero matcher.Mode
			if zero != matcher.Fatal {
				t.Fatalf("the zero Mode is %v, want Fatal", zero)
			}
		})
	})
}
