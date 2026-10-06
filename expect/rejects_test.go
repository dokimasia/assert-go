// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// rejection keeps the records that a call of Rejects returns.
var rejection []expect.Failure

// TestRejects runs the shared cases of the assertion that a check fails,
// and the case of the recording surface.
func TestRejects(t *testing.T) {
	t.Parallel()

	t.Run("Rejects", func(t *testing.T) {
		t.Parallel()
		matchertest.RunRejects(t, func(s *matchertest.Seat, msg string, check func(matcher.Seat)) []matcher.Failure {
			return expect.Rejects(s, msg, func(tb assert.TB) { check(tb) })
		})

		t.Run("records a record of rejects through Errorf when the driven check passes", func(t *testing.T) {
			t.Parallel()

			outer := &matchertest.Seat{}
			got := expect.Rejects(outer, "rejects 1", func(tb assert.TB) {
				expect.Equal(tb, 1, 1, "the value is one")
			})

			if len(got) != 0 {
				t.Fatalf("returned %+v, want no record of a check that passed", got)
			}
			if len(outer.Errs()) != 1 || len(outer.Fatals()) != 0 {
				t.Fatalf("reported %q through Errorf and %q through Fatalf, want one failure that lets the test run on",
					outer.Errs(), outer.Fatals())
			}
		})
	})
}

// TestRejectsAllocs checks the allocation ceiling of a passing call of
// Rejects.
func TestRejectsAllocs(t *testing.T) {
	alloctest.Check(t, rejectsCases())
}

// BenchmarkRejects measures a passing call of Rejects.
func BenchmarkRejects(b *testing.B) {
	for _, c := range rejectsCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// rejectsCases returns a passing call of Rejects of a check that fails at
// a call of True, with its allocation ceiling, measured.
func rejectsCases() []alloctest.Case {
	check := func(tb assert.TB) { expect.True(tb, false, "the check fails") }
	return []alloctest.Case{
		{Name: "Rejects", Call: func(tb assert.TB) { rejection = expect.Rejects(tb, allocContract, check) }, Allocs: 8},
	}
}
