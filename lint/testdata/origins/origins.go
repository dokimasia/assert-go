// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package origins

import (
	"encoding/json"
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type state struct{ target string }

func read(t *testing.T, path string, s *state) {
	defined, err := os.Readlink(path)
	assert.NoError(t, err, "the link reads")
	assert.Equal(t, defined, "target", "the link names its target") // want `links-to: state the check with files.LinksTo`
	var assigned string
	assigned, _ = os.Readlink(path)
	expect.Equal(t, assigned, "target", "the link names its target") // want `links-to: state the check with files.LinksTo`
	var declared, _ = os.Readlink(path)
	assert.Equal(t, declared, "target", "the link names its target") // want `links-to: state the check with files.LinksTo`
	once := string(defined)
	expect.Equal(t, once, "target", "the link names its target") // want `links-to: state the check with files.LinksTo`
	twice := string([]byte(once))
	assert.Equal(t, twice, "target", "the conversions keep the target")
	s.target, _ = os.Readlink(path)
	expect.Equal(t, s.target, "target", "the state holds the target")
}

func decoded(t *testing.T, v map[string]int) {
	data, _ := json.Marshal(v)
	var back map[string]int
	_ = json.Unmarshal(data, &back)
	assert.Equal(t, back, v, "decoding undoes encoding") // want `round-trip: state the check with RoundTrip of json\.Marshal and json\.Unmarshal`
}
