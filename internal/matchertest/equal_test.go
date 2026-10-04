// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matchertest"
)

// argsOf returns the arguments of the case named name in cases.
func argsOf(t *testing.T, cases []matchertest.Case, name string) []any {
	t.Helper()
	for _, c := range cases {
		if c.Name == name {
			return c.Args
		}
	}
	t.Fatalf("no case is named %q", name)
	return nil
}

// TestEqual checks the shape of the cases of equal and of not-equal, and
// that the arguments of a case state what its name claims.
func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("EqualCases", func(t *testing.T) {
		t.Parallel()

		t.Run("states a table of the shape that every runner assumes", func(t *testing.T) {
			t.Parallel()
			checkTable(t, "EqualCases", matchertest.EqualCases(), 2)
		})

		t.Run("states two times that their Equal method equates", func(t *testing.T) {
			t.Parallel()
			args := argsOf(t, matchertest.EqualCases(), "times that their Equal method equates differ by their fields")
			if !args[0].(time.Time).Equal(args[1].(time.Time)) {
				t.Fatal("the times of the case are two instants, want one in two locations")
			}
		})

		t.Run("states two closures of one function literal that return other values", func(t *testing.T) {
			t.Parallel()
			args := argsOf(t, matchertest.EqualCases(), "two closures of one function literal compare by their code")
			if args[0].(func() int)() == args[1].(func() int)() {
				t.Fatal("the closures of the case return one value, want two")
			}
		})
	})

	t.Run("NotEqualCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "NotEqualCases", matchertest.NotEqualCases(), 2)
	})
}
