// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propcontainsinorder

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.ContainsInOrder(t, subject.Name, []string{"n"}, "every name starts with n")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every name starts with n", func(c *prop.Case) { // want `property-form: state the check with prop.ContainsInOrder`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.ContainsInOrder(c, subject.Name(n), []string{"n"}, "every name starts with n")
	})
}
