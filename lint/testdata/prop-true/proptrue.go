// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package proptrue

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.True(t, subject.Valid, "every number is valid")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every number is valid", func(c *prop.Case) { // want `property-form: state the check with prop.True`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.True(c, subject.Valid(n), "every number is valid")
	})
}

func discarded(t *testing.T) {
	prop.ForAll(t, "zero is valid for every draw", func(c *prop.Case) { // want `property-form: state the check with prop.True`
		_ = c.Draw(prop.Integer(-1000, 1000), "n")
		assert.True(c, subject.Valid(0), "zero is valid for every draw")
	})
}
