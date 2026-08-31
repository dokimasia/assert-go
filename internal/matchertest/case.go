// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
	// Assertion is the canonical id the failure must carry. Read only
	// when Fails is true, and an empty value checks nothing.
	Assertion string
	// Detail is what the failure's record must hold. Every field
	// stated must match; a field left out is not checked. Read only
	// when Fails is true.
	Detail map[string]any
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
			checkOutcome(t, seat, tc)
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
func Verdict(seat *Seat, want Case) error {
	if !want.Fails {
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

	records := seat.Records()
	if len(records) == 0 {
		return errors.New("matchertest: reported no record; the assertion did not report through Fail")
	}
	first := records[0]

	if first.Contract != contractMsg {
		return fmt.Errorf("matchertest: record carries contract %q, want %q",
			first.Contract, contractMsg)
	}
	if want.Assertion != "" && first.Assertion != want.Assertion {
		return fmt.Errorf("matchertest: record names assertion %q, want %q",
			first.Assertion, want.Assertion)
	}
	for name, value := range want.Detail {
		held, ok := first.Detail[name]
		if !ok {
			return fmt.Errorf("matchertest: record holds no detail %q, want %+v", name, value)
		}
		if !cmp.Equal(held, value, cmpopts.EquateErrors()) {
			return fmt.Errorf("matchertest: detail %q is %+v, want %+v", name, held, value)
		}
	}
	return nil
}

// checkOutcome fails t when the seat's outcome is not what the case
// required.
func checkOutcome(t *testing.T, seat *Seat, tc Case) {
	t.Helper()

	if err := Verdict(seat, tc); err != nil {
		t.Fatal(err)
	}
}
