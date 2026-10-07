// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propcommutative

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Commutative(t, subject.Add, "addition commutes")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "addition commutes", func(c *prop.Case) { // want `property-form: state the check with prop.Commutative`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Commutative(c, subject.Add, n, 1, "addition commutes")
	})
}
