// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package eventuallytrue

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
)

func ready() bool { return true }

func correct(t *testing.T) {
	assert.EventuallyTrue(t, time.Second, ready, "the server becomes ready within a second")
}

func polled(t *testing.T) {
	limit := time.Now().Add(time.Second)
	for !ready() { // want `eventually: state the check with EventuallyTrue or Eventually`
		if time.Now().After(limit) {
			t.Fatal("the server is not ready within a second")
		}
		time.Sleep(10 * time.Millisecond)
	}
	deadline := time.Now().Add(time.Second)
	for _, attempt := range []int{1, 2, 3} { // want `eventually: state the check with EventuallyTrue or Eventually`
		if ready() || time.Now().After(deadline) {
			break
		}
		t.Log(attempt)
		time.Sleep(time.Duration(attempt) * time.Millisecond)
	}
}
