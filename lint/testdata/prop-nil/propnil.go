// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propnil

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Nil(t, subject.Pointer, "no number has a pointer")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "no number has a pointer", func(c *prop.Case) { // want `property-form: state the check with prop.Nil`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Nil(c, subject.Pointer(n), "no number has a pointer")
	})
}
