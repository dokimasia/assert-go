// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// contractMsg is what every runner passes as the caller's message.
// Declared here rather than exported, so a drift between the two
// shows up as a failing case in this file.
const contractMsg = "the stated contract"

// checkTable holds a case table to the shape every runner assumes. A
// table that drifts from it yields a suite that passes having checked
// less than it claims, which is the failure these tables exist to
// prevent one level down.
func checkTable(t *testing.T, name string, cases []matchertest.Case, arity int) {
	t.Helper()

	if len(cases) == 0 {
		t.Fatalf("%s is empty; its suite would pass having checked nothing", name)
	}

	seen := map[string]bool{}
	passing, failing := 0, 0

	for _, tc := range cases {
		switch {
		case tc.Name == "":
			t.Errorf("%s has a case with no name", name)
		case seen[tc.Name]:
			t.Errorf("%s repeats the case name %q", name, tc.Name)
		}
		seen[tc.Name] = true

		if got := len(tc.Args); got < arity {
			t.Errorf("%s case %q carries %d args, want at least %d", name, tc.Name, got, arity)
		}

		if tc.Fails {
			failing++
			continue
		}

		passing++
		if len(tc.Detail) > 0 || tc.Assertion != "" {
			t.Errorf("%s case %q passes but states what a failure would hold", name, tc.Name)
		}
	}

	if passing == 0 {
		t.Errorf("%s has no passing case; it would not notice an assertion that always fails", name)
	}
	if failing == 0 {
		t.Errorf("%s has no failing case; it would not notice an assertion that never fails", name)
	}
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
				t.Fatal("Verdict accepted a record holding none of the stated want")
			}
		})

		t.Run("a record holding a different value is rejected", func(t *testing.T) {
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
