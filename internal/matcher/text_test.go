// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestText runs the shared cases of the text assertions.
func TestText(t *testing.T) {
	t.Parallel()

	t.Run("HasPrefix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasPrefixCases(),
			func(s *matchertest.Seat, got, prefix any, msg string) {
				matcher.HasPrefix(s, matcher.Fatal, got, prefix.(string), msg)
			})
	})

	t.Run("HasSuffix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasSuffixCases(),
			func(s *matchertest.Seat, got, suffix any, msg string) {
				matcher.HasSuffix(s, matcher.Fatal, got, suffix.(string), msg)
			})
	})

	t.Run("Matches", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.MatchesCases(),
			func(s *matchertest.Seat, got, pattern any, msg string) {
				matcher.Matches(s, matcher.Fatal, got, pattern.(string), msg)
			})
	})
}

// TestTextAllocs checks the allocation ceiling of a passing call of each
// text assertion.
func TestTextAllocs(t *testing.T) {
	checkAllocs(t, textCases())
}

// BenchmarkText measures a passing call of each text assertion.
func BenchmarkText(b *testing.B) {
	benchAllocs(b, textCases())
}

// textCases returns a passing call of each text assertion on a string,
// with its allocation ceiling, measured.
func textCases() []allocCase {
	return []allocCase{
		{name: "HasPrefix", call: func(seat matcher.Seat) {
			matcher.HasPrefix(seat, matcher.Fatal, "store: missing", "store: ", allocContract)
		}},
		{name: "HasSuffix", call: func(seat matcher.Seat) {
			matcher.HasSuffix(seat, matcher.Fatal, "store: missing", "missing", allocContract)
		}},
		{name: "Matches", allocs: 62, call: func(seat matcher.Seat) {
			matcher.Matches(seat, matcher.Fatal, "order 42", `^order \d+$`, allocContract)
		}},
	}
}
