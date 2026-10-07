// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pure

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type status struct{ count int }

type store struct{}

func (store) Get(key string) string { return "" }

func (store) Snapshot() map[string]string { return nil }

func (store) Status() status { return status{} }

type reader struct{}

func (reader) Read(p []byte) (int, error) { return 0, nil }

func classify(err error) string { return "" }

func correct(t *testing.T, s store) {
	assert.Pure(t, s.Snapshot, func() { s.Get("key") }, "Get leaves the store as it was")
}

func observed(t *testing.T, s store) {
	before := s.Snapshot()
	s.Get("key")
	assert.Equal(t, s.Snapshot(), before, "Get leaves the store as it was") // want `pure: state the check with Pure of s\.Snapshot\(\), around s\.Get\("key"\)`
	first := s.Snapshot()
	s.Get("other")
	t.Log("read the other key")
	second := s.Snapshot()
	expect.Equal(t, first, second, "Get leaves the store as it was") // want `pure: state the check with Pure of s\.Snapshot\(\), around s\.Get\("other"\); t\.Log\("read the other key"\)`
}

func described(t *testing.T, s store) {
	first := s.Snapshot()
	description := "the store after a read of the key " + s.Get("key") + " and nothing else"
	second := s.Snapshot()
	assert.Equal(t, second, first, description) // want `pure: state the check with Pure of s\.Snapshot\(\), around description := "the store after a read of the key " \+ s\.Get…$`
}

func changed(t *testing.T, s store) {
	snapshot := s.Snapshot()
	snapshot["key"] = "value"
	s.Get("key")
	assert.Equal(t, s.Snapshot(), snapshot, "Get leaves the store as it was")
	state := s.Status()
	state.count = 2
	s.Get("key")
	assert.Equal(t, s.Status(), state, "Get leaves the status as it was")
}

func refused(t *testing.T, r reader) {
	_, first := r.Read(make([]byte, 64))
	assert.Equal(t, classify(first), "integrity", "the first read returns the verdict")
	n, again := r.Read(make([]byte, 64))
	expect.Equal(t, n, 0, "a reader after its verdict yields no byte")
	expect.Equal(t, again, first, "every read after the verdict returns the same error")
}

func built(t *testing.T, s store) {
	a := make([]int, 3)
	s.Get("key")
	b := make([]int, 3)
	assert.Equal(t, a, b, "the buffers start alike")
	x := []byte("key")
	s.Get("key")
	y := []byte("key")
	assert.Equal(t, x, y, "the keys start alike")
}
