// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package enumtest_test

import (
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/enumtest"
)

// colour is an enumeration of two members for the checks to examine.
type colour uint8

// The members of colour.
const (
	red colour = iota
	green
)

// pastColours is the first value past the members of colour.
const pastColours = colour(2)

// Valid reports whether c is red or green.
func (c colour) Valid() bool { return c <= green }

// String returns the name of a member, and colour(n) for any other value.
func (c colour) String() string {
	switch c {
	case red:
		return "red"
	case green:
		return "green"
	}
	return "colour(" + strconv.Itoa(int(c)) + ")"
}

// TestEnum checks that each check passes for an enumeration that meets it,
// and fails for one that does not.
func TestEnum(t *testing.T) {
	t.Parallel()

	t.Run("Members", func(t *testing.T) {
		t.Parallel()

		t.Run("passes for valid members and others that are not valid", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			enumtest.Members(rec, []colour{red, green}, []colour{pastColours})
			assert.False(t, rec.Failed(), "the check passes")
			assert.True(t, rec.HelperCalls() > 0, "the check marks its own frame")
		})

		t.Run("fails for a member that is not valid", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			enumtest.Members(rec, []colour{red, pastColours}, nil)
			failures := rec.Failures()
			assert.Length(t, failures, 1, "one member is not valid")
			assert.Equal(t, []string{failures[0].Assertion, failures[0].Contract},
				[]string{"true", "colour(2) is a member"}, "the check names the member")
		})

		t.Run("fails for another value that is valid", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			enumtest.Members(rec, []colour{red}, []colour{green})
			failures := rec.Failures()
			assert.Length(t, failures, 1, "one other value is valid")
			assert.Equal(t, []string{failures[0].Assertion, failures[0].Contract},
				[]string{"false", "green is no member"}, "the check names the value")
		})
	})

	t.Run("Spellings", func(t *testing.T) {
		t.Parallel()

		t.Run("passes for values that spell as stated", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			enumtest.Spellings(rec, map[colour]string{red: "red", green: "green", pastColours: "colour(2)"})
			assert.False(t, rec.Failed(), "the check passes")
			assert.True(t, rec.HelperCalls() > 0, "the check marks its own frame")
		})

		t.Run("fails for a value that spells otherwise", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			enumtest.Spellings(rec, map[colour]string{green: "Green"})
			failures := rec.Failures()
			assert.Length(t, failures, 1, "one spelling differs")
			assert.Equal(t, failures[0].Contract, "the spelling of Green", "the check names the spelling")
			assert.Equal(t, failures[0].Detail, map[string]any{"want": "Green", "got": "green"},
				"the check reports both spellings")
		})
	})
}
