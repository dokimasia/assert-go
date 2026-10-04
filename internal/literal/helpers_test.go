// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package literal_test

// celsius is a defined type over an integer, whose literal is the integer.
type celsius int

// point is a struct with exported and unexported fields.
type point struct {
	// X is exported, so the literal states it.
	X int
	// label is unexported, so the literal leaves it out.
	label string
	// Tags are exported, so the literal states them.
	Tags []bool
}
