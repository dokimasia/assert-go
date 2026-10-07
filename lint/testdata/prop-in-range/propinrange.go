// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propinrange

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.InRange(t, subject.Ratio, 0, 1, "every ratio is a fraction")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every ratio is a fraction", func(c *prop.Case) { // want `property-form: state the check with prop.InRange`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.InRange(c, subject.Ratio(n), 0, 1, "every ratio is a fraction")
	})
}
