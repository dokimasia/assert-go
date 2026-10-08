// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package notpure

import (
	"testing"

	"go.dokimi.dev/assert"
)

type store struct{}

func (store) Put(key string) {}

func (store) Snapshot() map[string]string { return nil }

func (store) Counts() []int { return nil }

func correct(t *testing.T, s store) {
	assert.NotPure(t, s.Snapshot, func() { s.Put("key") }, "Put changes the store")
}

func observed(t *testing.T, s store) {
	before := s.Snapshot()
	s.Put("key")
	assert.NotEqual(t, s.Snapshot(), before, "Put changes the store") // want `not-pure: state the check with NotPure of s\.Snapshot\(\), around s\.Put\("key"\)`
}

func copied(t *testing.T, s store) {
	counts := s.Counts()
	counts[0]++
	assert.NotEqual(t, counts, s.Counts(), "a change to a copy leaves the store's own")
	cleared := s.Counts()
	clear(cleared)
	assert.NotEqual(t, cleared, s.Counts(), "a change to a copy leaves the store's own")
	filled := s.Counts()
	copy(filled, []int{9})
	assert.NotEqual(t, filled, s.Counts(), "a change to a copy leaves the store's own")
}

func double(counts []int) {}

func written(t *testing.T, s store) {
	counts := s.Counts()
	double(counts)
	assert.NotEqual(t, counts, s.Counts(), "double writes the counts") // want `not-pure: state the check with NotPure of a copy of counts, around double\(counts\)$`
}
