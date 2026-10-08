// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package proplength

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.Length(t, subject.Items, 1, "every list holds one item")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every list holds one item", func(c *prop.Case) { // want `property-form: state the check with prop.Length`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Length(c, subject.Items(n), 1, "every list holds one item")
	})
}
