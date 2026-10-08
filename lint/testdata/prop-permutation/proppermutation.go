// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package proppermutation

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.IsPermutation(t, subject.Items, subject.Items, "a list is a permutation of itself")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "a list is a permutation of itself", func(c *prop.Case) { // want `property-form: state the check with prop.IsPermutation`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Permutation(c, subject.Items(n), subject.Items(n), "a list is a permutation of itself")
	})
}
