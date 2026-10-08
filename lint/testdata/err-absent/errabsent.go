// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package errabsent

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type coded interface {
	error
	Code() int
}

func open() error { return nil }

func correct(t *testing.T, err error) {
	assert.NoError(t, err, "the call succeeds")
}

func compared(t *testing.T, err error, c coded) {
	assert.True(t, err == nil, "the call succeeds")  // want `error-nil: state the check with NoError`
	expect.False(t, nil != err, "the call succeeds") // want `error-nil: state the check with NoError`
	assert.True(t, c == nil, "the call succeeds")    // want `error-nil: state the check with NoError`
	if err != nil {                                  // want `error-nil: state the check with NoError`
		t.Fatal(err)
	}
	if err := open(); err != nil { // want `error-nil: state the check with NoError`
		t.Fatalf("open: %v", err)
	}
}

func equalled(t *testing.T, err error) {
	assert.Equal(t, err, nil, "the call succeeds") // want `equal-nil: state the check with NoError`
}
