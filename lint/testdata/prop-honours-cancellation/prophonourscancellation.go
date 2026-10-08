// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prophonourscancellation

import (
	"context"
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.HonoursCancellation(t, subject.Fetch, "Fetch stops when its context is cancelled")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "Fetch stops when its context is cancelled", func(c *prop.Case) { // want `property-form: state the check with prop.HonoursCancellation`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.HonoursCancellation(c, func(ctx context.Context) error { return subject.Fetch(ctx, n) },
			"Fetch stops when its context is cancelled")
	})
}
