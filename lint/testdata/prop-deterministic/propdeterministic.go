// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propdeterministic

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Deterministic(t, subject.Encode, "the text of a number is one text")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "the text of a number is one text", func(c *prop.Case) { // want `property-form: state the check with prop.Deterministic`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Deterministic(c, subject.Encode, n, "the text of a number is one text")
	})
}
