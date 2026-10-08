// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package propforall

import (
	"math/rand/v2"
	"testing"
	"testing/quick"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/prop"
)

func correct(t *testing.T) {
	prop.ForAll(t, "every number encodes and is valid", func(c *prop.Case) {
		n := c.Draw(prop.Integer(0, 1000), "n")
		_, err := subject.Encode(n)
		assert.NoError(c, err, "every number encodes")
		expect.True(c, subject.Valid(n), "every number is valid")
	})
	prop.ForAll(t, "every pair adds", func(c *prop.Case) {
		a := c.Draw(prop.Integer(0, 1000), "a")
		b := c.Draw(prop.Integer(0, 1000), "b")
		assert.True(c, subject.Valid(subject.Add(a, b)), "every sum is valid")
	})
	prop.ForAll(t, "one draw and a log", func(c *prop.Case) {
		n := c.Draw(prop.Integer(0, 1000), "n")
		c.Logf("drew %d", n)
	})
	prop.ForAll(t, "a body of two assertions", func(c *prop.Case) {
		assert.True(c, true, "the case runs")
		assert.True(c, true, "the case runs again")
	})
	prop.ForAll(t, "a property without a form", func(c *prop.Case) {
		n := c.Draw(prop.Integer(0, 1000), "n")
		assert.EventuallyTrue(c, 0, func() bool { return subject.Valid(n) }, "the number becomes valid")
	})
	prop.ForAll(t, "a body written elsewhere", body)
}

func body(c *prop.Case) {
	assert.True(c, true, "the case runs")
}

func random(t *testing.T) {
	for range 100 { // want `for-all: state the check with prop.ForAll`
		n := rand.IntN(1000)
		assert.True(t, subject.Valid(n), "every number is valid")
	}
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 100; i++ { // want `for-all: state the check with prop.ForAll`
		if !subject.Valid(r.IntN(1000)) { // want `condition: state the check with True`
			t.Fatal("a number is invalid")
		}
	}
	assert.NoError(t, quick.Check(subject.Valid, nil), "every number is valid") // want `for-all: state the check with prop.ForAll`
	for range 3 {
		t.Log(rand.IntN(10))
	}
}
