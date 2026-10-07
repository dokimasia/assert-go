// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propidempotent

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	s := &subject.Store{}
	prop.Idempotent(t, s.Put, s.Snapshot, "a repeated Put leaves the store as one Put left it")
}

func handWritten(t *testing.T) {
	s := &subject.Store{}
	prop.ForAll(t, "a repeated Put leaves the store as one Put left it", func(c *prop.Case) { // want `property-form: state the check with prop.Idempotent`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Idempotent(c, s.Put, n, s.Snapshot, "a repeated Put leaves the store as one Put left it")
	})
}
