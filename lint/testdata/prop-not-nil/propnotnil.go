// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propnotnil

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.NotNil(t, subject.Pointer, "every number has a pointer")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every number has a pointer", func(c *prop.Case) { // want `property-form: state the check with prop.NotNil`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.NotNil(c, subject.Pointer(n), "every number has a pointer")
	})
}
