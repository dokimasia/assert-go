// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propfalse

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.False(t, subject.Valid, "no number is valid")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "no number is valid", func(c *prop.Case) { // want `property-form: state the check with prop.False`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.False(c, subject.Valid(n), "no number is valid")
	})
}
