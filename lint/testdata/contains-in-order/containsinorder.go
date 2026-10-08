// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package containsinorder

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func correct(t *testing.T, s string) {
	assert.ContainsInOrder(t, s, []string{"name", "age"}, "the name comes before the age")
}

func ordered(t *testing.T, s, other string) {
	assert.True(t, strings.Index(s, "name") < strings.Index(s, "age"), "the name comes before the age") // want `contains-in-order: state the check with ContainsInOrder`
	expect.True(t, strings.Index(s, "age") > strings.Index(s, "name"), "the age comes after the name")  // want `contains-in-order: state the check with ContainsInOrder`
	if strings.Index(s, "name") >= strings.Index(s, "age") {                                            // want `contains-in-order: state the check with ContainsInOrder`
		t.Fatal("the age comes first")
	}
	assert.True(t, strings.Index(s, "name") < strings.Index(other, "age"), "the name comes first")
	assert.True(t, strings.Index(s, "name") < 3, "the name is near the start") // want `order: state the check with InRange`
}
