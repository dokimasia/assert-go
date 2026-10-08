// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propnotthrows

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.NotPanics(t, subject.Work, "Work returns for every number")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "Work returns for every number", func(c *prop.Case) { // want `property-form: state the check with prop.NotPanics`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.NotPanics(c, func() { subject.Work(n) }, "Work returns for every number")
	})
}
