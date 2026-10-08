// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package notnil

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, p *int) {
	assert.NotNil(t, p, "the pointer is present")
}

func compared(t *testing.T, p *int, m map[string]int, ch chan int) {
	assert.False(t, m == nil, "the map is present")     // want `nil: state the check with NotNil`
	expect.True(t, ch != nil, "the channel is present") // want `nil: state the check with NotNil`
	if p == nil {                                       // want `nil: state the check with NotNil`
		t.Fatal("the pointer is nil")
	}
}

func equalled(t *testing.T, p *int) {
	assert.NotEqual(t, p, nil, "the pointer is present") // want `equal-nil: state the check with NotNil`
}
