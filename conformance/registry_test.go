// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Registry", func(t *testing.T) {
		t.Parallel()

		t.Run("every registered assertion is one the definition states", func(t *testing.T) {
			t.Parallel()

			assertions, err := conformance.Assertions()
			assert.NoError(t, err, "the assertion table can be read")

			for id := range conformance.Registry {
				assert.Contains(t, assertions, id,
					"the definition states "+string(id))
			}
		})

		t.Run("every registered assertion has corpus cases", func(t *testing.T) {
			t.Parallel()

			byAssertion, err := conformance.Cases()
			assert.NoError(t, err, "the corpus can be read")

			for id := range conformance.Registry {
				assert.Contains(t, byAssertion, id,
					"the corpus covers "+string(id))
			}
		})

		t.Run("an invoker drives the assertion it names", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			conformance.Registry["equal"](r, []any{1, 2}, "the values match")

			assert.True(t, r.Failed(), "the equal invoker reports on differing values")
		})
	})
}
