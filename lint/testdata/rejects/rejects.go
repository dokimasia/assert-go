// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package rejects

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type job struct{}

func (job) Failed() bool { return false }

func refusesADuplicate(tb assert.TB) {
	assert.True(tb, false, "the key was already present")
}

func correct(t *testing.T) {
	got := assert.Rejects(t, "a store that overwrites fails the check", refusesADuplicate)
	assert.Length(t, got, 1, "the check fails once")
}

func recorded(t *testing.T, j job) {
	assert.True(t, j.Failed(), "the job failed")
	rec := assert.NewRecorder()
	refusesADuplicate(rec)
	assert.True(t, rec.Failed(), "a store that overwrites fails the check") // want `rejects: state the check with Rejects`
	if !rec.Failed() {                                                      // want `rejects: state the check with Rejects`
		t.Fatal("the check passes a store that overwrites")
	}
	expect.False(t, rec.Failed(), "the check passes")
	assert.True(t, t.Failed(), "the test has failed")
}
