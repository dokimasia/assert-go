// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matches

import (
	"regexp"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

var id = regexp.MustCompile(`^usr_[0-9]+$`)

func correct(t *testing.T, s string) {
	assert.Matches(t, s, `^usr_\d+$`, "the id is a user's")
}

func matched(t *testing.T, s string, b []byte) {
	assert.True(t, id.MatchString(s), "the id is a user's")                     // want `matches: state the check with Matches of id\.MatchString\(s\)`
	expect.True(t, regexp.MustCompile(`^\{`).Match(b), "the body is an object") // want `matches: state the check with Matches`
	ok, err := regexp.MatchString(`^usr_`, s)
	assert.NoError(t, err, "the pattern compiles")
	assert.True(t, ok, "the id is a user's") // want `matches: state the check with Matches`
	if !id.MatchString(s) {                  // want `matches: state the check with Matches`
		t.Fatalf("the id %q is no user's", s)
	}
	assert.False(t, id.MatchString(s), "the id is no user's")
}
