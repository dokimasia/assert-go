// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package edits

import (
	"bytes"
	"testing"

	"go.dokimi.dev/assert"
	e "go.dokimi.dev/assert/expect"
)

func rewritten(t *testing.T, p *int, xs []int, a, b []byte) {
	assert.True(t, // want `nil: state the check with Nil`
		p == nil,
		"the pointer is nil",
	)
	assert.Equal(t, len(func() []int { return xs }()), 0, "the closure returns no item") // want `length: state the check with Empty`
	assert.True(t, len([]int{1, 2}) == 2, "the literal has two items")                   // want `length: state the check with Length`
	e.True(t, bytes.Equal(a, b), "the bodies match")                                     // want `equal-func: state the check with Equal and EquateEmpty`
	assert.True(t, p /* the pointer */ == nil, "the pointer is nil")                     // want `nil: state the check with Nil`
}
