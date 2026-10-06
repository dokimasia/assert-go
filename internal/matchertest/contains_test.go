// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// permutes reports whether got and want contain the same elements, each as
// often, by removing each element of got from a copy of want. Two empty
// slices match when both are nil or neither is, or when opts equate empty
// values.
func permutes(got, want []any, opts []matcher.Option) bool {
	if len(got) == 0 && len(want) == 0 {
		return (got == nil) == (want == nil) || rulesOf(opts).EquateEmpty
	}
	pool := append([]any(nil), want...)
	for _, g := range got {
		found := -1
		for j, w := range pool {
			if same(g, w, opts) {
				found = j
				break
			}
		}
		if found < 0 {
			return false
		}
		pool = append(pool[:found], pool[found+1:]...)
	}
	return len(pool) == 0
}

func TestContains(t *testing.T) {
	t.Parallel()

	t.Run("ContainsCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "ContainsCases", matchertest.ContainsCases(), 2)
	})

	t.Run("NotContainsCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "NotContainsCases", matchertest.NotContainsCases(), 2)
	})

	t.Run("ContainsInOrderCases", func(t *testing.T) {
		t.Parallel()
		checkTable(t, "ContainsInOrderCases", matchertest.ContainsInOrderCases(), 2)
	})

	// The twin implements permutation independently of the package under
	// test, and drives it through its suite. A suite that no implementation
	// passes looks like coverage and checks nothing.
	t.Run("RunPermutation", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPermutation(t, func(s *matchertest.Seat, got, want []any, msg string,
			opts ...matcher.Option,
		) {
			if !permutes(got, want, opts) {
				report(s, "permutation", msg, map[string]any{"want": want, "got": got})
			}
		})
	})
}

// TestContainsTwins runs TestContainsTwinsChild in a child process, and
// requires the failure of RunPermutation for a twin that reports a field
// that permutation does not declare.
func TestContainsTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestContainsTwinsChild",
		`the record contains the fields ["got" "index" "want"], want ["got" "want"]`)
}

// TestContainsTwinsChild runs only in the child process of
// TestContainsTwins.
func TestContainsTwinsChild(t *testing.T) {
	inChild(t)

	t.Run("RunPermutation of a twin that reports an undeclared field", func(t *testing.T) {
		matchertest.RunPermutation(t, func(s *matchertest.Seat, got, want []any, msg string, opts ...matcher.Option) {
			if !permutes(got, want, opts) {
				report(s, "permutation", msg, map[string]any{"want": want, "got": got, "index": 0})
			}
		})
	})
}
