// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package throws

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func explode() { panic("boom") }

func correct(t *testing.T) {
	assert.Panics(t, explode, "the call panics")
}

func deferredIf(t *testing.T) {
	defer func() {
		if r := recover(); r == nil { // want `panics: state the check with Panics`
			t.Fatal("the call does not panic")
		}
	}()
	explode()
}

func deferredAssertion(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover(), "the call panics") // want `panics: state the check with Panics`
	}()
	explode()
}

func deferredCondition(t *testing.T) {
	defer func() {
		r := recover()
		expect.True(t, r != nil, "the call panics") // want `panics: state the check with Panics`
	}()
	explode()
}

func deferredOthers(t *testing.T, cleanup func()) {
	defer cleanup()
	defer func() {
		t.Log("the call ended")
	}()
	defer func() {
		if recover() != "boom" { // want `condition: state the check with False`
			t.Fatal("the call panics with another value")
		}
	}()
	explode()
}
