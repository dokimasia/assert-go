// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// contractMsg is what every runner passes as the caller's message. It is
// declared here and not exported, so a drift between the two fails a
// case in this file.
const contractMsg = "the stated contract"

// errStated is the error that a case states, which a wrapped error
// matches.
var errStated = errors.New("matchertest_test: stated")

// pair is a struct with an unexported field.
type pair struct {
	A int
	b int
}

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

// TestCase checks the verdict that every runner of a shared suite ends in.
func TestCase(t *testing.T) {
	t.Parallel()

	t.Run("Verdict", func(t *testing.T) {
		t.Parallel()

		t.Run("accepts a passing case that reported nothing", func(t *testing.T) {
			t.Parallel()

			if err := matchertest.Verdict(&matchertest.Seat{}, matchertest.Case{}); err != nil {
				t.Fatalf("Verdict = %v, want nil", err)
			}
		})

		t.Run("rejects a passing case that reported a failure", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{Assertion: "equal", Contract: contractMsg}, true)

			if err := matchertest.Verdict(s, matchertest.Case{}); err == nil {
				t.Fatal("Verdict accepted a failure where the case expected none")
			}
		})

		t.Run("rejects a failing case that reported nothing", func(t *testing.T) {
			t.Parallel()

			if err := matchertest.Verdict(&matchertest.Seat{}, matchertest.Case{Fails: true}); err == nil {
				t.Fatal("Verdict accepted silence where the case expected a failure")
			}
		})

		t.Run("rejects a failure that does not lead with the caller's message", func(t *testing.T) {
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

		t.Run("rejects a record without a stated detail field", func(t *testing.T) {
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

		t.Run("rejects a failure without a record", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("%s: a sentence without a record", contractMsg)

			err := matchertest.Verdict(s, matchertest.Case{Fails: true})
			if err == nil || !strings.Contains(err.Error(), "no record") {
				t.Fatalf("Verdict = %v, want it to name the missing record", err)
			}
		})

		t.Run("rejects a record of another contract", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Fatalf("%s: the sentence of the failure", contractMsg)
			s.Report(matcher.Failure{Assertion: "equal", Contract: "another contract"}, true)

			err := matchertest.Verdict(s, matchertest.Case{Fails: true})
			if err == nil || !strings.Contains(err.Error(), "another contract") {
				t.Fatalf("Verdict = %v, want it to name the contract of the record", err)
			}
		})

		t.Run("rejects a record of another value", func(t *testing.T) {
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

		t.Run("rejects a record of another assertion", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{Assertion: "not-equal", Contract: contractMsg}, true)

			want := matchertest.Case{Fails: true, Assertion: "equal"}
			if err := matchertest.Verdict(s, want); err == nil {
				t.Fatal("Verdict accepted a record naming an assertion the case did not")
			}
		})

		t.Run("accepts a record that matches the case", func(t *testing.T) {
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

		t.Run("accepts a failure that a recording surface reported", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			s.Report(matcher.Failure{Assertion: "equal", Contract: contractMsg}, false)

			if err := matchertest.Verdict(s, matchertest.Case{Fails: true}); err != nil {
				t.Fatalf("Verdict = %v; a recording surface reports through Errorf", err)
			}
		})

		t.Run("compares a value of the detail with the case's", func(t *testing.T) {
			t.Parallel()

			nan := math.NaN()
			one, other := 1, 1
			entries, empty, fields := map[string]int{"a": 1}, map[string]int{}, pair{A: 1, b: 2}
			tests := []struct {
				name      string
				giveHeld  any
				giveValue any
				want      bool
			}{
				{
					name:      "matches an error that wraps the stated error",
					giveHeld:  fmt.Errorf("wrapped: %w", errStated),
					giveValue: errStated,
					want:      true,
				},
				{
					name:      "matches an error that the stated error wraps",
					giveHeld:  errStated,
					giveValue: fmt.Errorf("wrapped: %w", errStated),
					want:      true,
				},
				{name: "matches no error of another chain", giveHeld: errors.New("x"), giveValue: errStated},
				{name: "matches a NaN with a NaN", giveHeld: nan, giveValue: nan, want: true},
				{
					name:      "matches a NaN inside a list with a NaN",
					giveHeld:  []any{nan},
					giveValue: []any{nan},
					want:      true,
				},
				{name: "matches no NaN with a number", giveHeld: nan, giveValue: 1.0},
				{
					name:      "matches a *big.Int of the stated value",
					giveHeld:  big.NewInt(5),
					giveValue: big.NewInt(5),
					want:      true,
				},
				{name: "matches no *big.Int of another value", giveHeld: big.NewInt(5), giveValue: big.NewInt(6)},
				{
					name:      "matches a nil *big.Int with a nil one",
					giveHeld:  (*big.Int)(nil),
					giveValue: (*big.Int)(nil),
					want:      true,
				},
				{name: "matches no nil *big.Int with a value", giveHeld: (*big.Int)(nil), giveValue: big.NewInt(0)},
				{name: "matches no value of another type", giveHeld: 1, giveValue: int64(1)},
				{name: "matches no nil with a value", giveHeld: nil, giveValue: 1},
				{name: "matches pointers to equal values", giveHeld: &one, giveValue: &other, want: true},
				{name: "matches no nil pointer with a pointer", giveHeld: (*int)(nil), giveValue: &one},
				{name: "matches no nil slice with an empty one", giveHeld: []int(nil), giveValue: []int{}},
				{name: "matches no slice of another length", giveHeld: []int{1}, giveValue: []int{1, 2}},
				{name: "matches no slice of another element", giveHeld: []int{1, 2}, giveValue: []int{1, 3}},
				{name: "matches arrays of equal elements", giveHeld: [2]int{1, 2}, giveValue: [2]int{1, 2}, want: true},
				{name: "matches no array of another element", giveHeld: [2]int{1, 2}, giveValue: [2]int{1, 3}},
				{
					name:      "matches maps of equal entries",
					giveHeld:  entries,
					giveValue: map[string]int{"a": 1},
					want:      true,
				},
				{name: "matches no nil map with an empty one", giveHeld: map[string]int(nil), giveValue: empty},
				{name: "matches no map of another length", giveHeld: entries, giveValue: empty},
				{name: "matches no map of another key", giveHeld: entries, giveValue: map[string]int{"b": 1}},
				{name: "matches no map of another value", giveHeld: entries, giveValue: map[string]int{"a": 2}},
				{name: "matches structs of equal fields", giveHeld: fields, giveValue: pair{A: 1, b: 2}, want: true},
				{name: "matches no struct of another unexported field", giveHeld: fields, giveValue: pair{A: 1, b: 3}},
				{name: "matches two nil functions", giveHeld: (func())(nil), giveValue: (func())(nil), want: true},
				{name: "matches no function that is not nil", giveHeld: func() {}, giveValue: func() {}},
				{name: "matches equal strings", giveHeld: "a", giveValue: "a", want: true},
				{name: "matches no other string", giveHeld: "a", giveValue: "b"},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()

					s := &matchertest.Seat{}
					s.Report(matcher.Failure{
						Assertion: "equal", Contract: contractMsg, Detail: map[string]any{"v": tt.giveHeld},
					}, true)
					err := matchertest.Verdict(s, matchertest.Case{
						Fails: true, Detail: map[string]any{"v": tt.giveValue},
					})
					if (err == nil) != tt.want {
						t.Fatalf("Verdict = %v, want a match: %v", err, tt.want)
					}
				})
			}
		})
	})
}
