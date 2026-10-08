// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package proppairwise

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func ascending(earlier, later int) bool { return earlier <= later }

func correct(t *testing.T) {
	prop.Pairwise(t, subject.Items, ascending, "every list is sorted")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "every list is sorted", func(c *prop.Case) { // want `property-form: state the check with prop.Pairwise`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.Pairwise(c, subject.Items(n), ascending, "every list is sorted")
	})
}
