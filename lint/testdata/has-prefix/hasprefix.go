// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package hasprefix

import (
	"bytes"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, id string) {
	assert.HasPrefix(t, id, "usr_", "the id starts with its prefix")
}

func prefixed(t *testing.T, id string, b, p []byte) {
	assert.True(t, strings.HasPrefix(id, "usr_"), "the id starts with its prefix")  // want `has-prefix: state the check with HasPrefix`
	expect.True(t, bytes.HasPrefix(b, []byte("{")), "the body starts with a brace") // want `has-prefix: state the check with HasPrefix`
	assert.True(t, bytes.HasPrefix(b, p), "the body starts with the preamble")      // want `has-prefix: state the check with HasPrefix`
	if !strings.HasPrefix(id, "usr_") {                                             // want `has-prefix: state the check with HasPrefix`
		t.Fatalf("the id %q lacks its prefix", id)
	}
	assert.False(t, strings.HasPrefix(id, "tmp_"), "the id is no temporary one")
}
