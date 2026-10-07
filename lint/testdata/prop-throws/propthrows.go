// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propthrows

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Panics(t, subject.Explode, "Explode panics for every number")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "Explode panics for every number", func(c *prop.Case) { // want `property-form: state the check with prop.Panics`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Panics(c, func() { subject.Explode(n) }, "Explode panics for every number")
	})
}
