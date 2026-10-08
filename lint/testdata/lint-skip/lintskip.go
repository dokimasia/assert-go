// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lintskip

import (
	"testing"

	"go.dokimi.dev/assert"
)

func annotated(t *testing.T, items []int) {
	//dokimi:lint-skip conjunction: the annotation covers the next line
	assert.True(t, len(items) > 0 && items[0] == 1, "the first item is one")
	assert.True(t, len(items) > 0 && items[0] == 1, "the first item is one") //dokimi:lint-skip conjunction: after code
	assert.True(t, len(items) > 0 && items[0] == 1, "the first item is one") // want `conjunction: state each operand`
}

func unused(t *testing.T, items []int) {
	//dokimi:lint-skip compare , conjunction: no report of compare is on the next line // want `lint-skip: remove compare, because no report of the rule starts on the line that the annotation covers`
	assert.True(t, len(items) > 0 && items[0] == 1, "the first item is one")
	//dokimi:lint-skip conjunction: the next line is blank // want `lint-skip: remove conjunction`

	assert.True(t, len(items) > 0 && items[0] == 1, "the first item is one") // want `conjunction: state each operand`
}

func malformed(t *testing.T, items []int) {
	//dokimi:lint-skip conjunction // want "state the rules and a reason after them"
	// want +1 "state the rules and a reason after them"
	//dokimi:lint-skip conjunction:
	// want +1 "state the rules and a reason after them"
	//dokimi:lint-skip
	//dokimi:lint-skip conjunction,: an empty rule // want "state the rules and a reason after them"
	//dokimi:lint-skipped conjunction: no annotation
	assert.True(t, len(items) > 0 && items[0] == 1, "the first item is one") // want `conjunction: state each operand`
}
