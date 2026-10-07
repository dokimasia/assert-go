// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propmatches

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Matches(t, subject.Name, `^n-?\d+$`, "every name is n and a number")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every name is n and a number", func(c *prop.Case) { // want `property-form: state the check with prop.Matches`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Matches(c, subject.Name(n), `^n-?\d+$`, "every name is n and a number")
	})
}
