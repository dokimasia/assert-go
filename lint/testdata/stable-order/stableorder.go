// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stableorder

import (
	"testing"

	"go.dokimi.dev/assert"
)

type store struct{}

func (store) Keys() []string { return nil }

func (store) List() ([]string, error) { return nil, nil }

func (store) Digest() []byte { return nil }

type verdict uint8

func (store) Verdicts() []verdict { return nil }

func correct(t *testing.T, s store) {
	assert.StableOrder(t, s.List, "the store lists its keys in one order")
}

func repeated(t *testing.T, s store) {
	first := s.Keys()
	second := s.Keys()
	assert.Equal(t, second, first, "the store lists its keys in one order") // want `stable-order: state the check with StableOrder of s\.Keys\(\)`
	one := s.Digest()
	two := s.Digest()
	assert.Equal(t, two, one, "the digest of the store is one digest") // want `deterministic: state the check with Deterministic`
	before := s.Verdicts()
	after := s.Verdicts()
	assert.Equal(t, after, before, "the store lists its verdicts in one order") // want `stable-order: state the check with StableOrder of s\.Verdicts\(\)`
}
