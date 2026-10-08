// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propmaxallocs

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.MaxAllocs(t, subject.Work, 0, "Work allocates nothing")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "Work allocates nothing", func(c *prop.Case) { // want `property-form: state the check with prop.MaxAllocs`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.MaxAllocs(c, func() { subject.Work(n) }, 0, "Work allocates nothing")
	})
}
