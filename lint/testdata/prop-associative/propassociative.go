// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propassociative

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Associative(t, subject.Add, "addition associates")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "addition associates", func(c *prop.Case) { // want `property-form: state the check with prop.Associative`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Associative(c, subject.Add, n, 1, 2, "addition associates")
	})
}
