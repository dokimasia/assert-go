// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// contractMsg is what every runner passes as the caller's message. It is
// declared here and not exported, so a drift between the two fails a
// case in this file.
const contractMsg = "the stated contract"

// TestCaseTwins runs TestCaseTwinsChild in a child process, and requires
// the failure of a suite over no cases.
func TestCaseTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestCaseTwinsChild", "the suite has no cases; it would pass having checked nothing")
}

// TestCaseTwinsChild runs only in the child process of TestCaseTwins.
func TestCaseTwinsChild(t *testing.T) {
	inChild(t)

	t.Run("RunOne over no cases", func(t *testing.T) {
		matchertest.RunOne(t, nil, func(*matchertest.Seat, any, string) {})
	})
}

func TestCase(t *testing.T) {
	t.Parallel()

	t.Run("Verdict", func(t *testing.T) {
		t.Parallel()

		t.Run("a passing case with no failure is accepted", func(t *testing.T) {
			t.Parallel()

			if err := matchertest.Verdict(&matchertest.Seat{}, matchertest.Case{}); err != nil {
				t.Fatalf("Verdict = %v, want nil", err)
			}
		})

		t.Run("a passing case that reported is rejected", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{Assertion: "equal", Contract: contractMsg}, true)

			if err := matchertest.Verdict(s, matchertest.Case{}); err == nil {
				t.Fatal("Verdict accepted a failure where the case expected none")
			}
		})

		t.Run("a failing case with no failure is rejected", func(t *testing.T) {
			t.Parallel()

			if err := matchertest.Verdict(&matchertest.Seat{}, matchertest.Case{Fails: true}); err == nil {
				t.Fatal("Verdict accepted silence where the case expected a failure")
			}
		})

		t.Run("a failure not leading with the caller's message is rejected", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("a message that does not lead correctly")

			err := matchertest.Verdict(s, matchertest.Case{Fails: true})
			if err == nil {
				t.Fatal("Verdict accepted a failure that dropped the caller's message")
			}
			if !strings.Contains(err.Error(), "lead") {
				t.Fatalf("Verdict = %v, want it to name the missing message", err)
			}
		})

		t.Run("a record missing a stated detail field is rejected", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{
				Assertion: "equal", Contract: contractMsg,
				Detail: map[string]any{"got": 2},
			}, true)

			want := matchertest.Case{
				Fails: true, Assertion: "equal",
				Detail: map[string]any{"want": 1, "got": 2},
			}
			if err := matchertest.Verdict(s, want); err == nil {
				t.Fatal("Verdict accepted a record without the stated want")
			}
		})

		t.Run("a failure without a record is rejected", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("%s: a sentence without a record", contractMsg)

			err := matchertest.Verdict(s, matchertest.Case{Fails: true})
			if err == nil || !strings.Contains(err.Error(), "no record") {
				t.Fatalf("Verdict = %v, want it to name the missing record", err)
			}
		})

		t.Run("a record of another contract is rejected", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("%s: the sentence of the failure", contractMsg)
			s.Report(matcher.Failure{Assertion: "equal", Contract: "another contract"}, true)

			err := matchertest.Verdict(s, matchertest.Case{Fails: true})
			if err == nil || !strings.Contains(err.Error(), "another contract") {
				t.Fatalf("Verdict = %v, want it to name the contract of the record", err)
			}
		})

		t.Run("a record with a different value is rejected", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{
				Assertion: "equal", Contract: contractMsg,
				Detail: map[string]any{"want": 9, "got": 2},
			}, true)

			want := matchertest.Case{
				Fails: true, Assertion: "equal",
				Detail: map[string]any{"want": 1, "got": 2},
			}
			if err := matchertest.Verdict(s, want); err == nil {
				t.Fatal("Verdict accepted a record whose want differs from the case")
			}
		})

		t.Run("a record naming a different assertion is rejected", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{Assertion: "not-equal", Contract: contractMsg}, true)

			want := matchertest.Case{Fails: true, Assertion: "equal"}
			if err := matchertest.Verdict(s, want); err == nil {
				t.Fatal("Verdict accepted a record naming an assertion the case did not")
			}
		})

		t.Run("a record matching the case is accepted", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{
				Assertion: "equal", Contract: contractMsg,
				Detail: map[string]any{"want": 1, "got": 2},
			}, true)

			want := matchertest.Case{
				Fails: true, Assertion: "equal",
				Detail: map[string]any{"want": 1, "got": 2},
			}
			if err := matchertest.Verdict(s, want); err != nil {
				t.Fatalf("Verdict = %v, want nil", err)
			}
		})

		t.Run("a recorded failure counts as a failure", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{Assertion: "equal", Contract: contractMsg}, false)

			if err := matchertest.Verdict(s, matchertest.Case{Fails: true}); err != nil {
				t.Fatalf("Verdict = %v; a recording surface reports through Errorf", err)
			}
		})
	})
}
