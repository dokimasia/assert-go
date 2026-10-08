// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package poisoned

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type store struct{}

func (store) Corrupt() {}

func (store) Read() error { return nil }

func correct(t *testing.T, s store) {
	assert.Poisoned(t, s.Corrupt, s.Read, "a corrupt store fails every read")
}

func induced(t *testing.T, s store, err error) {
	s.Corrupt()
	for range 32 { // want `poisoned: state the check with Poisoned`
		assert.HasError(t, s.Read(), "a corrupt store fails every read")
	}
	for i := 0; i < 32; i++ { // want `poisoned: state the check with Poisoned`
		expect.HasError(t, s.Read(), "a corrupt store fails every read")
	}
	for {
		assert.HasError(t, s.Read(), "a corrupt store fails every read")
		break
	}
	for range 32 {
		assert.HasError(t, err, "the error is present")
	}
	for range 32 {
		assert.NoError(t, s.Read(), "a sound store passes every read")
	}
	for range 32 {
		t.Log("read")
	}
	for range 32 {
		assert.HasError(t, s.Read(), "a corrupt store fails every read")
		s.Corrupt()
	}
}
