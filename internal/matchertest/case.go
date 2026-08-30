// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Case is one assertion's inputs and the outcome every surface must
// produce from them.
//
// Args holds what the assertion is given after the seat, in call
// order. A runner reads as many as its arity needs, so one Case type
// serves assertions of different shapes.
type Case struct {
	// Name says what the case establishes, and becomes the subtest
	// name.
	Name string
	// Args are the assertion's arguments after the seat, excluding the
	// trailing message.
	Args []any
	// Fails says whether the assertion must report.
	Fails bool
	// Contains are substrings the failure must carry. Read only when
	// Fails is true.
	Contains []string
}

// Invoke1 calls an assertion taking one value.
type Invoke1 func(seat *Seat, got any, msg string)

// Invoke2 calls an assertion taking two values.
type Invoke2 func(seat *Seat, got, want any, msg string)

// Invoke3 calls an assertion taking three values.
type Invoke3 func(seat *Seat, got, second, third any, msg string)

// RunOne drives invoke against every case, reading one argument.
func RunOne(t *testing.T, cases []Case, invoke Invoke1) {
	t.Helper()
	run(t, cases, func(seat *Seat, args []any, msg string) {
		invoke(seat, args[0], msg)
	})
}

// RunPair drives invoke against every case, reading two arguments.
func RunPair(t *testing.T, cases []Case, invoke Invoke2) {
	t.Helper()
	run(t, cases, func(seat *Seat, args []any, msg string) {
		invoke(seat, args[0], args[1], msg)
	})
}

// RunTriple drives invoke against every case, reading three arguments.
func RunTriple(t *testing.T, cases []Case, invoke Invoke3) {
	t.Helper()
	run(t, cases, func(seat *Seat, args []any, msg string) {
		invoke(seat, args[0], args[1], args[2], msg)
	})
}

// contractMsg is the message every case passes, so a runner can check
// that a failure leads with it.
const contractMsg = "the stated contract"

// run drives call against every case and checks the outcome.
func run(t *testing.T, cases []Case, call func(seat *Seat, args []any, msg string)) {
	t.Helper()

	if len(cases) == 0 {
		t.Fatal("the suite has no cases; it would pass having checked nothing")
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			seat := &Seat{}
			call(seat, tc.Args, contractMsg)
			checkOutcome(t, seat, tc.Fails, tc.Contains)
		})
	}
}

// Verdict reports how a seat's outcome differs from what a case
// required, and nil when it matches.
//
// Every runner ends here, so one verdict covers assertions of every
// shape and no suite invents its own idea of what passing means. It
// returns an error rather than failing a test so the rule itself can
// be tested: a checker that only ever calls t.Fatal cannot be driven
// against a case it should reject.
func Verdict(seat *Seat, fails bool, contains []string) error {
	if !fails {
		if seat.Failed() {
			return fmt.Errorf("matchertest: reported %q, want nothing", seat.First())
		}
		return nil
	}

	if !seat.Failed() {
		return errors.New("matchertest: reported nothing, want a failure")
	}

	got := seat.First()
	if !strings.HasPrefix(got, contractMsg) {
		return fmt.Errorf("matchertest: failure %q does not lead with the caller's message", got)
	}
	for _, want := range contains {
		if !strings.Contains(got, want) {
			return fmt.Errorf("matchertest: failure %q does not carry %q", got, want)
		}
	}
	return nil
}

// checkOutcome fails t when the seat's outcome is not what the case
// required.
func checkOutcome(t *testing.T, seat *Seat, fails bool, contains []string) {
	t.Helper()

	if err := Verdict(seat, fails, contains); err != nil {
		t.Fatal(err)
	}
}
