// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package properrpresent

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.HasError(t, subject.Check, "every number fails the check")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every number fails the check", func(c *prop.Case) { // want `property-form: state the check with prop.HasError`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.HasError(c, subject.Check(n), "every number fails the check")
	})
}
