// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/record"
)

// TestDefinition checks the constant that go generate writes from the
// vendored definition.
func TestDefinition(t *testing.T) {
	t.Parallel()

	t.Run("Definition", func(t *testing.T) {
		t.Parallel()

		t.Run("is the version of the vendored definition", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, record.Definition, conformance.Version(),
				"every call record states the version of the vendored definition")
		})
	})
}
