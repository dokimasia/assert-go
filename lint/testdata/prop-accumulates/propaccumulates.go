// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propaccumulates

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	s := &subject.Store{}
	prop.Accumulates(t, s.Put, s.Count, "each Put adds one value")
}

func handWritten(t *testing.T) {
	s := &subject.Store{}
	prop.ForAll(t, "each Put adds one value", func(c *prop.Case) { // want `property-form: state the check with prop.Accumulates`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Accumulates(c, s.Put, n, s.Count, "each Put adds one value")
	})
}
