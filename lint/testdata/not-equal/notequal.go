// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package notequal

import (
	"bytes"
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, token, previous string) {
	assert.NotEqual(t, token, previous, "Refresh issues a new token")
}

func compared(t *testing.T, n int, f float64, s string) {
	assert.False(t, n == 3, "the count is not three")   // want `compare: state the check with NotEqual`
	expect.True(t, f != 0.5, "the ratio is not a half") // want `compare: state the check with NotEqual`
	assert.True(t, !(s == "x"), "the name is not x")    // want `compare: state the check with NotEqual`
	if s == "x" {                                       // want `compare: state the check with NotEqual`
		t.Fatal("the name is x")
	}
}

func deep(t *testing.T, a, b []int, x, y any, p, q []byte) {
	assert.False(t, reflect.DeepEqual(a, b), "the stores hold two lists") // want `deep-equal: state the check with NotEqual`
	expect.False(t, reflect.DeepEqual(x, y), "the values differ")         // want `deep-equal: state the check with NotEqual`
	assert.False(t, bytes.Equal(p, q), "the bodies differ")               // want `equal-func: state the check with NotEqual and EquateEmpty`
	if reflect.DeepEqual(a, b) {                                          // want `deep-equal: state the check with NotEqual`
		t.Fatal("the stores hold one list")
	}
}
