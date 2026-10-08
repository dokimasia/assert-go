// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package nil

import (
	"io"
	"testing"
	"unsafe"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type pathError struct{}

func (*pathError) Error() string { return "path" }

func correct(t *testing.T, p *int) {
	assert.Nil(t, p, "the pointer is nil")
}

func compared(t *testing.T, p *int, s []int, f func(), u unsafe.Pointer, e *pathError) {
	assert.True(t, p == nil, "the pointer is nil")       // want `nil: state the check with Nil`
	assert.True(t, nil == s, "the slice is nil")         // want `nil: state the check with Nil`
	expect.True(t, !(f != nil), "the function is nil")   // want `nil: state the check with Nil`
	assert.False(t, (u != nil), "the pointer is nil")    // want `nil: state the check with Nil`
	assert.True(t, e == nil, "the error pointer is nil") // want `nil: state the check with Nil`
	if s != nil {                                        // want `nil: state the check with Nil`
		t.Fatal("the slice is present")
	}
}

func equalled(t *testing.T, p *int, m map[string]int, r io.Reader, s []int) {
	assert.Equal(t, p, nil, "the pointer is nil") // want `equal-nil: state the check with Nil`
	expect.Equal(t, nil, m, "the map is nil")     // want `equal-nil: state the check with Nil`
	assert.Equal(t, r, nil, "the reader is nil")
	assert.Equal(t, s, nil, "the slice is empty", assert.EquateEmpty())
}

func interfaces(t *testing.T, r io.Reader) {
	assert.True(t, r == nil, "the reader is nil")
}

func typed[P interface {
	*pathError
	error
}](t *testing.T, p P) {
	assert.True(t, p == nil, "the error is nil")
}
