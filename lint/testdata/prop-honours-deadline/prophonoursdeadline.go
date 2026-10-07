// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prophonoursdeadline

import (
	"context"
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.HonoursDeadline(t, subject.Fetch, "Fetch stops when its deadline passes")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "Fetch stops when its deadline passes", func(c *prop.Case) { // want `property-form: state the check with prop.HonoursDeadline`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.HonoursDeadline(c, func(ctx context.Context) error { return subject.Fetch(ctx, n) },
			"Fetch stops when its deadline passes")
	})
}
