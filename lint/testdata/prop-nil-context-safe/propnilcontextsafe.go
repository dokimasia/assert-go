// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propnilcontextsafe

import (
	"context"
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.NilContextSafe(t, subject.Fetch, "Fetch accepts a nil context")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "Fetch accepts a nil context", func(c *prop.Case) { // want `property-form: state the check with prop.NilContextSafe`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.NilContextSafe(c, func(ctx context.Context) error { return subject.Fetch(ctx, n) },
			"Fetch accepts a nil context")
	})
}
