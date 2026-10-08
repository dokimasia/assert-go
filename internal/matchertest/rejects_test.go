// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestRejects(t *testing.T) {
	t.Parallel()

	t.Run("RunRejects", func(t *testing.T) {
		t.Parallel()

		matchertest.RunRejects(t, func(s *matchertest.Seat, msg string, check func(matcher.Seat)) []matcher.Failure {
			return matcher.Rejects(s, matcher.Fatal, msg, check)
		})
	})
}

// TestRejectsTwins runs TestRejectsTwinsProcess in a child process, and
// requires the failures of RunRejects for a twin that never runs the check,
// a twin that runs the check past each failure and keeps only the contracts
// of its records, and a twin that returns a record of a check that passes.
func TestRejectsTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestRejectsTwinsProcess",
		"returned [], want the records of both failures in call order",
		"returned [], want the record of the failure that stopped the check",
		"returned [] and ran on: false, want no record and a check that Errorf let run",
		"matchertest: reported nothing, want a failure",
		"want one of equal that states got 3 and want 1",
		"the check ran past a failure that stops it",
		"returned [] and ran on: true, want no record and a check that Fatalf stopped",
		"and rejects declares no detail field",
		"want no record of a check that passed")
}

// TestRejectsTwinsProcess runs only in the child process of TestRejectsTwins.
func TestRejectsTwinsProcess(t *testing.T) {
	inChild(t)

	t.Run("RunRejects of a twin that never runs the check", func(t *testing.T) {
		matchertest.RunRejects(t, func(*matchertest.Seat, string, func(matcher.Seat)) []matcher.Failure { return nil })
	})

	t.Run("RunRejects of a twin that runs the check past each failure and keeps its contracts", func(t *testing.T) {
		matchertest.RunRejects(t, func(s *matchertest.Seat, msg string, check func(matcher.Seat)) []matcher.Failure {
			inner := &matchertest.Seat{}
			check(inner)
			if !inner.Failed() {
				passed := map[string]any{"passed": true}
				s.Report(matcher.Failure{Assertion: "rejects", Contract: msg, Detail: passed}, true)
				return nil
			}
			var contracts []matcher.Failure
			for _, f := range inner.Records() {
				contracts = append(contracts, matcher.Failure{Contract: f.Contract})
			}
			return contracts
		})
	})

	t.Run("RunRejects of a twin that returns a record of a check that passes", func(t *testing.T) {
		matchertest.RunRejects(t, func(s *matchertest.Seat, msg string, check func(matcher.Seat)) []matcher.Failure {
			got := matcher.Rejects(s, matcher.Fatal, msg, check)
			return append(got, matcher.Failure{Assertion: "rejects", Contract: msg})
		})
	})
}
