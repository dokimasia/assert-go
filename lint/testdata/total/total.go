// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package total

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type checker struct{}

func (checker) Check(name string) error { return nil }

func valid(name string) error { return nil }

func registry() checker { return checker{} }

func check(t *testing.T, name string) error {
	t.Helper()
	return nil
}

func correct(t *testing.T, names []string) {
	assert.Total(t, valid, names, "every name is valid")
}

func looped(t *testing.T, names []string, c checker) {
	for _, name := range names { // want `total: state the check with Total`
		assert.NoError(t, valid(name), "every name is valid")
	}
	for _, name := range names { // want `total: state the check with Total`
		assert.NoError(t, c.Check(name), "every name passes the check")
	}
	for _, name := range names { // want `total: state the check with Total`
		expect.NoError(t, valid(name), "every name is valid")
	}
	for _, name := range names { // want `total: state the check with Total`
		assert.NoError(t, registry().Check(name), "every name passes the check of the registry")
	}
	for _, name := range names { // want `total: state the check with Total`
		assert.NoError(t, valid(name), "the name "+name+" is valid")
	}
}

func unlooped(t *testing.T, names []string, byName map[string]string, err error) {
	for _, name := range names {
		assert.NoError(t, check(t, name), "every name passes the check")
	}
	for _, name := range names {
		assert.NoError(t, valid(name+"_suffix"), "every name with its suffix is valid")
	}
	for i := range names {
		assert.NoError(t, valid(names[i]), "every name is valid")
	}
	for _, name := range names {
		assert.NoError(t, valid(name), "every name is valid")
		assert.NotEmpty(t, name, "every name is set")
	}
	for _, name := range byName {
		assert.NoError(t, valid(name), "every name is valid")
	}
	for _, name := range names {
		assert.Nil(t, valid(name), "every name is valid")
	}
	for _, name := range names {
		assert.NoError(t, valid("fixed"), name)
	}
	for _, name := range names {
		assert.NoError(t, err, name)
	}
	for _, name := range names {
		fmt.Println(valid(name))
	}
	for _, name := range names {
		_ = valid(name)
	}
}
