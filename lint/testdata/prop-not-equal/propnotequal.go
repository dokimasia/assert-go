// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propnotequal

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.NotEqual(t, subject.Double, subject.Twice, "doubling differs from adding")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "doubling differs from adding", func(c *prop.Case) { // want `property-form: state the check with prop.NotEqual`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.NotEqual(c, subject.Double(n), subject.Twice(n), "doubling differs from adding")
	})
}
