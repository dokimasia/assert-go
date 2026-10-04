// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package enumtest

import (
	"fmt"

	"go.dokimi.dev/assert"
)

// Members checks that Valid reports true for each of members and false for
// each of others, the values past the members.
func Members[E interface {
	Valid() bool
	fmt.Stringer
}](tb assert.TB, members, others []E,
) {
	tb.Helper()
	for _, m := range members {
		assert.True(tb, m.Valid(), m.String()+" is a member")
	}
	for _, o := range others {
		assert.False(tb, o.Valid(), o.String()+" is no member")
	}
}

// Spellings checks that String returns the spelling that want states for
// each of its values: the definition's spelling of each member, and for a
// value past the members, stringer's spelling of the type's name with the
// number in parentheses.
func Spellings[E interface {
	comparable
	fmt.Stringer
}](tb assert.TB, want map[E]string,
) {
	tb.Helper()
	for value, spelling := range want {
		assert.Equal(tb, value.String(), spelling, "the spelling of "+spelling)
	}
}
