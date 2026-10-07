// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package propnotpure

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	s := &subject.Store{}
	prop.NotPure(t, s.Snapshot, func(n int) { _ = s.Put(n) }, "Put changes the store")
}

func handWritten(t *testing.T) {
	s := &subject.Store{}
	prop.ForAll(t, "Put changes the store", func(c *prop.Case) { // want `property-form: state the check with prop.NotPure`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.NotPure(c, s.Snapshot, func() { _ = s.Put(n) }, "Put changes the store")
	})
}
