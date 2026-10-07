// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propcontains

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Contains(t, subject.Name, "n", "every name holds an n")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every name holds an n", func(c *prop.Case) { // want `property-form: state the check with prop.Contains`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Contains(c, subject.Name(n), "n", "every name holds an n")
	})
}

func needled(t *testing.T) {
	prop.ForAll(t, "every list holds its input", func(c *prop.Case) {
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Contains(c, subject.Items(n), n, "every list holds its input")
	})
}
