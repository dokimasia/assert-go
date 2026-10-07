// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package false

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func done() bool { return false }

func correct(t *testing.T, ok bool) {
	assert.False(t, ok, "the flag is clear")
	expect.False(t, done(), "the work is not done")
}

func ifChecks(t *testing.T, ok bool, flags []bool) {
	if ok { // want `condition: state the check with False`
		t.Fatal("the flag is set")
	}
	if done() { // want `condition: state the check with False`
		t.FailNow()
	}
	for _, flag := range flags {
		if flag { // want `condition: state the check with False`
			t.Fatal("a flag is set")
		}
	}
}
