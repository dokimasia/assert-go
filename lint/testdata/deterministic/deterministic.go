// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package deterministic

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func hash(s string) uint64 { return 0 }

func plan(s string) ([]string, error) { return nil, nil }

func correct(t *testing.T) {
	assert.Deterministic(t, plan, "input", "the plan of an input is one plan")
}

func repeated(t *testing.T, s string) {
	first, _ := plan(s)
	second, _ := plan(s)
	assert.Equal(t, second, first, "the plan of an input is one plan")    // want `deterministic: state the check with Deterministic`
	expect.Equal(t, hash(s), hash(s), "the hash of an input is one hash") // want `deterministic: state the check with Deterministic of hash\(s\)`
	if hash(s) != hash(s) {                                               // want `deterministic: state the check with Deterministic`
		t.Fatal("the hash changes")
	}
	a := hash(s)
	b := hash(s)
	assert.NotEqual(t, a, b, "the hashes differ")
	c := 3
	assert.Equal(t, c, 3, "the count is three")
}

func checked(t *testing.T, s string) {
	one, err := plan(s)
	assert.NoError(t, err, "the plan builds")
	two, err := plan(s)
	assert.NoError(t, err, "the plan builds again")
	assert.Equal(t, two, one, "the plan of an input is one plan") // want `deterministic: state the check with Deterministic`
}

func hashed(t *testing.T, s string) {
	one, _ := plan(s)
	assert.Equal(t, hash(s), uint64(0), "the hash of every input is zero")
	var label string
	two, _ := plan(s)
	assert.Equal(t, two, one, "the plan of "+label+" is one plan") // want `deterministic: state the check with Deterministic`
}

func failed(t *testing.T, s string) {
	_, first := plan(s)
	_, second := plan(s)
	assert.Equal(t, second, first, "the plan fails the same way twice")
}

type gauge struct{ n int }

func (g *gauge) Total() int { return g.n }

func count(t *testing.T, g *gauge) int { return g.n }

func prepare(t *testing.T) error { return nil }

func checkedTwice(t *testing.T, g *gauge) {
	first := g.Total()
	assert.Equal(t, g.Total(), first, "two readings agree") // want `deterministic: state the check with Deterministic of g\.Total\(\)$`
	one := count(t, g)
	assert.NoError(t, prepare(t), "the test is prepared")
	assert.Equal(t, count(t, g), one, "two counts agree") // want `deterministic: state the check with Deterministic of count\(t, g\)$`
}

func quietChecks(t *testing.T, g, other *gauge) {
	first := g.Total()
	assert.Equal(t, g.Total(), 0, "the gauge starts at zero")
	assert.Equal(t, g.Total(), first, "two readings agree") // want `deterministic: state the check with Deterministic of g\.Total\(\)$`
	second := g.Total()
	assert.NotPanics(t, func() { t.Logf("the gauge %v", g) }, "a log of the gauge is safe")
	assert.Equal(t, g.Total(), second, "two readings agree") // want `deterministic: state the check with Deterministic of g\.Total\(\)$`
	third := g.Total()
	assert.Equal(t, other.Total(), 3, "the other gauge reads three")
	assert.Equal(t, g.Total(), third, "two readings agree") // want `deterministic: state the check with Deterministic of g\.Total\(\)$`
}
