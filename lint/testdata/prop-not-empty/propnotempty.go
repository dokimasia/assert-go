// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propnotempty

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.NotEmpty(t, subject.Items, "every list holds an item")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every list holds an item", func(c *prop.Case) { // want `property-form: state the check with prop.NotEmpty`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.NotEmpty(c, subject.Items(n), "every list holds an item")
	})
}
