// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

// Compile-time proof that both seats of a test and the recorder satisfy TB.
var (
	_ expect.TB = (*testing.T)(nil)
	_ expect.TB = (*testing.B)(nil)
	_ expect.TB = (*expect.Recorder)(nil)
)

func TestTB(t *testing.T) {
	t.Parallel()

	t.Run("TB", func(t *testing.T) {
		t.Parallel()

		t.Run("is the seat of the aborting surface", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.TB](), reflect.TypeFor[assert.TB](), "one seat for both surfaces")
		})
	})
}
