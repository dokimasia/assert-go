// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propnotcontains

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.NotContains(t, subject.Items, 2000, "no list holds 2000")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "no list holds 2000", func(c *prop.Case) { // want `property-form: state the check with prop.NotContains`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.NotContains(c, subject.Items(n), 2000, "no list holds 2000")
	})
}
