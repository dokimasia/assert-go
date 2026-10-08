// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propequal

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Equal(t, subject.Double, subject.Twice, "doubling adds a number to itself")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "doubling adds a number to itself", func(c *prop.Case) { // want `property-form: state the check with prop.Equal`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Equal(c, subject.Double(n), subject.Twice(n), "doubling adds a number to itself")
	})
}
