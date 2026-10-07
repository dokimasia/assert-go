// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package idempotent

import (
	"testing"

	"go.dokimi.dev/assert"
)

type store struct{}

func (store) Put(key string) error { return nil }

func (store) List() []string { return nil }

func correct(t *testing.T, s store) {
	assert.Idempotent(t, s.Put, "key", s.List, "a repeated Put leaves the store as one Put left it")
}

func repeated(t *testing.T, s store) {
	_ = s.Put("key")
	once := s.List()
	_ = s.Put("key")
	twice := s.List()
	assert.Equal(t, twice, once, "a repeated Put leaves the store as one Put left it") // want `idempotent: state the check with Idempotent of s\.Put\("key"\), read by s\.List\(\)`
	_ = s.Put("other")
	again := s.List()
	assert.NotEqual(t, again, once, "a Put of another key changes the store") // want `not-pure: state the check with NotPure of s\.List\(\), around s\.Put\("key"\); twice := s\.List\(\); s\.Put\("other"\)`
}

func checked(t *testing.T, s store, stored bool) {
	_ = s.Put("key")
	assert.True(t, stored, "the store takes the key")
	once := s.List()
	_ = s.Put("key")
	assert.True(t, stored, "the store takes the key")
	twice := s.List()
	assert.Equal(t, twice, once, "a repeated Put leaves the store as one Put left it") // want `idempotent: state the check with Idempotent`
}
