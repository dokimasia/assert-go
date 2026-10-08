// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prophassuffix

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.HasSuffix(t, subject.Name, "0", "every name ends with zero")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every name ends with zero", func(c *prop.Case) { // want `property-form: state the check with prop.HasSuffix`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.HasSuffix(c, subject.Name(n), "0", "every name ends with zero")
	})
}
