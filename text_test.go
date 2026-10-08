// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestText runs the shared cases of the text assertions.
func TestText(t *testing.T) {
	t.Parallel()

	t.Run("HasPrefix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasPrefixCases(),
			func(s *matchertest.Seat, got, prefix any, msg string) {
				assert.HasPrefix(s, got, prefix.(string), msg)
			})
	})

	t.Run("HasSuffix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasSuffixCases(),
			func(s *matchertest.Seat, got, suffix any, msg string) {
				assert.HasSuffix(s, got, suffix.(string), msg)
			})
	})

	t.Run("Matches", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.MatchesCases(),
			func(s *matchertest.Seat, got, pattern any, msg string) {
				assert.Matches(s, got, pattern.(string), msg)
			})
	})
}

// TestTextAllocs checks the allocation ceiling of a passing call of each
// text assertion.
func TestTextAllocs(t *testing.T) {
	alloctest.Check(t, textCases())
}

// BenchmarkText measures a passing call of each text assertion.
func BenchmarkText(b *testing.B) {
	for _, c := range textCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// textCases returns a passing call of each text assertion on a string,
// with its allocation ceiling, measured.
func textCases() []alloctest.Case {
	return []alloctest.Case{
		{
			Name: "HasPrefix",
			Call: func(tb assert.TB) { assert.HasPrefix(tb, "store: missing", "store: ", allocContract) },
		},
		{
			Name: "HasSuffix",
			Call: func(tb assert.TB) { assert.HasSuffix(tb, "store: missing", "missing", allocContract) },
		},
		{
			Name:   "Matches",
			Call:   func(tb assert.TB) { assert.Matches(tb, "order 42", `^order \d+$`, allocContract) },
			Allocs: 62,
		},
	}
}
