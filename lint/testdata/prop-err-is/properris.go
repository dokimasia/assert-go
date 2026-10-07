// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package properris

import (
	"io"
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.ErrorIs(t, subject.Check, io.EOF, "every check ends the stream")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every check ends the stream", func(c *prop.Case) { // want `property-form: state the check with prop.ErrorIs`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.ErrorIs(c, subject.Check(n), io.EOF, "every check ends the stream")
	})
}
