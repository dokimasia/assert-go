// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package errpresent

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, err error) {
	assert.HasError(t, err, "the call fails")
}

func compared(t *testing.T, err error) {
	assert.True(t, err != nil, "the call fails") // want `error-nil: state the check with HasError`
	if err == nil {                              // want `error-nil: state the check with HasError`
		t.Fatal("the call succeeds")
	}
}

func equalled(t *testing.T, err error) {
	expect.NotEqual(t, err, nil, "the call fails") // want `equal-nil: state the check with HasError`
}
