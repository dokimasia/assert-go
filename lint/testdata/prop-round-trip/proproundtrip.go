// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package proproundtrip

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.RoundTrip(t, subject.Encode, subject.Decode, "decoding undoes encoding")
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "decoding undoes encoding", func(c *prop.Case) { // want `property-form: state the check with prop.RoundTrip`
		n := c.Draw(prop.Integer(-1000, 1000), "n")
		assert.RoundTrip(c, subject.Encode, subject.Decode, n, "decoding undoes encoding")
	})
}
