// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

// TestFailure checks that the record and the types beside it are the ones
// of the aborting surface.
func TestFailure(t *testing.T) {
	t.Parallel()

	t.Run("Failure", func(t *testing.T) {
		t.Parallel()

		t.Run("is the record of the aborting surface, with the call site and the reporter", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.Failure](), reflect.TypeFor[assert.Failure](), "one record")
			expect.Equal(t, reflect.TypeFor[expect.Where](), reflect.TypeFor[assert.Where](), "one call site")
			expect.Equal(t, reflect.TypeFor[expect.Reporter](), reflect.TypeFor[assert.Reporter](), "one reporter")
		})
	})
}
