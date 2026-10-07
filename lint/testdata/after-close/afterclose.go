// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package afterclose

import (
	"errors"
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

var errClosed = errors.New("afterclose: the store is closed")

type store struct{}

func (store) Close() error { return nil }

func (store) Get(key string) (string, error) { return "", nil }

func get(s store, key string) error { return nil }

func correct(t *testing.T, s store) {
	assert.FailsAfterClose(t, s.Close, func() error {
		_, err := s.Get("key")
		return err
	}, errClosed, "Get fails after Close")
}

func closed(t *testing.T, s store, f *os.File) {
	assert.NoError(t, s.Close(), "the store closes")
	_, err := s.Get("key")
	assert.ErrorIs(t, err, errClosed, "Get fails after Close") // want `after-close: state the check with FailsAfterClose of s\.Close\(\) and s\.Get\("key"\)`
	expect.HasError(t, err, "Get fails after Close")
	if err == nil { // want `error-nil: state the check with HasError`
		t.Fatal("Get succeeds after Close")
	}
	_ = f.Close()
	_, err = f.Write(nil)
	assert.True(t, errors.Is(err, os.ErrClosed), "Write fails after Close") // want `after-close: state the check with FailsAfterClose`
	_, werr := f.Write(nil)
	assert.True(t, werr == os.ErrClosed, "Write fails after Close") // want `sentinel: state the check with ErrorIs`
}

func others(t *testing.T, s, other store) {
	assert.NoError(t, s.Close(), "the store closes")
	_, err := other.Get("key")
	assert.ErrorIs(t, err, errClosed, "Get of another store fails")
	_ = s.Close()
	other = store{}
	_, err = s.Get("key")
	assert.ErrorIs(t, err, errClosed, "Get fails after Close")
	_ = s.Close()
	err = get(s, "key")
	assert.ErrorIs(t, err, errClosed, "Get fails after Close")
}

func open(t *testing.T, s store) {
	defer s.Close()
	_, err := s.Get("key")
	assert.ErrorIs(t, err, errClosed, "Get fails")
	expect.HasError(t, err, "Get fails")
}

func first(t *testing.T, s store) {
	_, err := s.Get("key")
	assert.ErrorIs(t, err, errClosed, "Get fails")
}

func unassigned(t *testing.T, s store, err error) {
	_ = s.Close()
	assert.ErrorIs(t, err, errClosed, "the error is the closed one")
}
