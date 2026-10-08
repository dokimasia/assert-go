// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propempty

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Empty(t, subject.Items, "every list is empty")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every list is empty", func(c *prop.Case) { // want `property-form: state the check with prop.Empty`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Empty(c, subject.Items(n), "every list is empty")
	})
}
