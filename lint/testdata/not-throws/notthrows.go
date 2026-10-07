// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package notthrows

import (
	"testing"

	"go.dokimi.dev/assert"
)

func work() {}

func correct(t *testing.T) {
	assert.NotPanics(t, work, "the call returns")
}

func deferredIf(t *testing.T) {
	defer func() {
		if r := recover(); r != nil { // want `not-panics: state the check with NotPanics`
			t.Fatalf("the call panics: %v", r)
		}
	}()
	work()
}

func deferredAssertion(t *testing.T) {
	defer func() {
		assert.Nil(t, recover(), "the call returns") // want `not-panics: state the check with NotPanics`
	}()
	work()
}
