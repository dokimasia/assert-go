// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package accumulates

import (
	"testing"

	"example.test/lint/subject"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T) {
	s := &subject.Store{}
	assert.Accumulates(t, s.Put, 1, s.Count, "each Put adds one value")
	expect.Accumulates(t, s.Put, 2, s.Count, "each Put adds one value")
}
