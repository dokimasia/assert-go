// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package completeswithin

import (
	"context"
	"testing"
	"time"

	"go.dokimi.dev/assert"
)

func work(ctx context.Context) error { return nil }

func correct(t *testing.T) {
	assert.CompletesWithin(t, time.Second, work, "the work completes within a second")
}

func selected(t *testing.T, done, stop chan struct{}) {
	select { // want `completes-within: state the check with CompletesWithin`
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the work does not complete within a second")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Log("the work is slow")
	}
	select {
	case <-done:
	default:
		t.Fatal("the work is not done")
	}
	select {
	case <-done:
	case <-stop:
		t.Fatal("the work stopped")
	}
}
