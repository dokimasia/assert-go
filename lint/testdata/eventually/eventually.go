// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package eventually

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
)

func ready() bool { return true }

func correct(t *testing.T) {
	assert.Eventually(t, time.Second, 10*time.Millisecond, func(tb assert.TB) {
		assert.True(tb, ready(), "the server is ready")
	}, "the server becomes ready within a second")
}

func polled(t *testing.T) {
	for i := 0; i < 100; i++ { // want `eventually: state the check with EventuallyTrue or Eventually`
		if ready() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for range 3 {
		t.Log("waiting")
	}
}
