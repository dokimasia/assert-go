// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package properras

import (
	"io/fs"
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.ErrorAs[*fs.PathError](t, subject.Check, "every failure is a path error")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every failure is a path error", func(c *prop.Case) { // want `property-form: state the check with prop.ErrorAs`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.ErrorAs[*fs.PathError](c, subject.Check(n), "every failure is a path error")
	})
}
