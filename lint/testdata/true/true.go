// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package true

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type point struct{ x int }

func valid(n int) bool { return n > 0 }

func correct(t *testing.T, ok bool, n int) {
	assert.True(t, ok, "the flag is set")
	expect.True(t, valid(n), "the count is valid")
}

func ifChecks(t *testing.T, ok bool, n int) {
	if !ok { // want `condition: state the check with True`
		t.Fatal("the flag is not set")
	}
	if !valid(n) { // want `condition: state the check with True`
		t.Errorf("the count %d is not valid", n)
	}
}

func conjunctions(t *testing.T, a, b, c bool, p *point, name string, names chan string) {
	assert.True(t, a && b, "both hold")                                   // want `conjunction: state each operand in an assertion of its own: True for a, True for b$`
	assert.True(t, a && (b && c), "all three hold")                       // want `conjunction: state each operand in an assertion of its own: True for a, True for b, True for c$`
	assert.False(t, !(a && b), "both hold")                               // want `conjunction: state each operand in an assertion of its own: True for a, True for b$`
	assert.True(t, p != nil && p.x > 0, "the point is right of the axis") // want `conjunction: state each operand in an assertion of its own: NotNil for p != nil, InRange for p\.x > 0$`
	assert.True(t, a && b, "case "+name)                                  // want `conjunction: state each operand in an assertion of its own`
	expect.True(t, p != nil && p.x > 0, "the point is right of the axis") // want `conjunction: state each operand in an assertion of its own: NotNil for p != nil, InRange for p\.x > 0$`
	defer assert.True(t, a && b, "both hold")                             // want `conjunction: state each operand in an assertion of its own`
	assert.True(t, a && b, fmt.Sprintf("case %s", name))                  // want `conjunction: state each operand in an assertion of its own`
	assert.True(t, a && b, <-names)                                       // want `conjunction: state each operand in an assertion of its own`
	if !(a && b) {                                                        // want `conjunction: state each operand in an assertion of its own: True for a, True for b$`
		t.Fatal("one of the two does not hold")
	}
	assert.False(t, a && b, "not both hold")
	assert.True(t, a || b, "one holds")
}
