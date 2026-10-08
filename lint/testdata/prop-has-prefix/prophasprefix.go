// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prophasprefix

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.HasPrefix(t, subject.Name, "n", "every name starts with n")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every name starts with n", func(c *prop.Case) { // want `property-form: state the check with prop.HasPrefix`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.HasPrefix(c, subject.Name(n), "n", "every name starts with n")
	})
}
