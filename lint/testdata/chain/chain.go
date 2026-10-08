// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package chain

import (
	"io"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	. "go.dokimi.dev/assert/expect"
)

func fixed(t *testing.T, xs []int, err error, n float64) {
	// want +1 `chain: state the assertions about xs with assert.That`
	assert.NotNil(t, xs, "the store returns its items")
	assert.Length(t, xs, 3, "the store returns three items")
	assert.Contains(t, xs, 2, "the store returns the second item", assert.EquateNaNs())

	// want +1 `chain: state the assertions about err with expect.That`
	expect.HasError(t, err, "the call fails")
	expect.ErrorIsNot(t, err, io.EOF, "the call does not end the stream")

	// want +1 `chain: state the assertions about n with assert.That`
	assert.InRange(t, n, 0, 1, "the ratio is a fraction")
	assert.CloseTo(t, n, 0.5, 0.5, "the ratio is near a half")
}

func reported(t *testing.T, xs []int, err error, p *int) {
	// want +1 `chain: state the assertions about xs with assert.That`
	assert.NotNil(t, xs, "the store returns its items")
	// The store returns three items.
	assert.Length(t, xs, 3, "the store returns three items")

	// want +1 `chain: state the assertions about err with expect.That`
	NoError(t, err, "the call succeeds")
	ErrorIsNot(t, err, io.EOF, "the call does not end the stream")

	// want +1 `chain: state the assertions about p with assert.That`
	assert.NotNil(t, p, "the pointer is present")
	assert.Equal(t, p, nil, "the pointer is nil") // want `equal-nil: state the check with Nil`
}

func unreported(t, u *testing.T, xs []int, s string, ok bool, done chan struct{}) {
	assert.NotNil(t, xs, "the store returns its items")
	assert.NotEmpty(t, s, "the name is set")

	assert.NotNil(t, xs, "the store returns its items")
	expect.Length(t, xs, 3, "the store returns three items")

	assert.True(t, ok, "the flag is set")
	assert.False(t, !ok, "the flag is set")

	assert.NotNil(t, xs, "the store returns its items")
	assert.NotNil(u, xs, "the store returns its items")

	assert.Equal(t, xs[0], 1, "the first item is one")
	assert.Equal(t, xs[1], 2, "the second item is two")

	<-done
	assert.NotNil(t, xs, "the store returns its items")
	_ = s
	assert.NotNil(t, xs, "the store returns its items")
	switch {
	case ok:
		assert.NotNil(t, xs, "the store returns its items")
	}
}
