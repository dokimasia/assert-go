// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propmaxallocswithsetup

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.MaxAllocsWithSetup(t, subject.Buffer, subject.Consume, 0, "Consume allocates nothing beside its buffer")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "Consume allocates nothing beside its buffer", func(c *prop.Case) { // want `property-form: state the check with prop.MaxAllocsWithSetup`
		n := c.Draw(prop.Integer(0, 1000), "n")
		assert.MaxAllocsWithSetup(c, func() []byte { return subject.Buffer(n) }, subject.Consume, 0,
			"Consume allocates nothing beside its buffer")
	})
}
