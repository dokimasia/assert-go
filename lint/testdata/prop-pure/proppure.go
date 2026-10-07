// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package proppure

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	s := &subject.Store{}
	prop.Pure(t, s.Snapshot, s.Get, "Get leaves the store as it was")
}

func handWritten(t *testing.T) {
	s := &subject.Store{}
	prop.ForAll(t, "Get leaves the store as it was", func(c *prop.Case) { // want `property-form: state the check with prop.Pure`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Pure(c, s.Snapshot, func() { s.Get(n) }, "Get leaves the store as it was")
	})
}

func observed(t *testing.T) {
	prop.ForAll(t, "Work leaves the list of its input as it was", func(c *prop.Case) {
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Pure(c, func() []int { return subject.Items(n) }, func() { subject.Work(n) },
			"Work leaves the list of its input as it was")
	})
}
