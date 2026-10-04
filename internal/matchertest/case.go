// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

// Case is one assertion's inputs and the outcome every surface must
// produce from them.
//
// Args are what the assertion is given after the seat, in call order. A
// runner reads as many as its arity needs, so one Case type serves
// assertions of different shapes.
type Case struct {
	// Name states what the case establishes, and becomes the subtest
	// name.
	Name string
	// Args are the assertion's arguments after the seat, excluding the
	// trailing message.
	Args []any
	// Fails is whether the assertion must report.
	Fails bool
	// Assertion is the canonical id that the failure's record must name.
	// Read only when Fails is true, and an empty value checks nothing.
	Assertion string
	// Detail is what the failure's record must contain. Every field
	// stated must match, and a field left out is not checked. Read only
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
		return fmt.Errorf("matchertest: record states contract %q, want %q",
			first.Contract, contractMsg)
	}
	if want.Assertion != "" && first.Assertion != want.Assertion {
		return fmt.Errorf("matchertest: record names assertion %q, want %q",
			first.Assertion, want.Assertion)
	}
	for name, value := range want.Detail {
		held, ok := first.Detail[name]
		if !ok {
			return fmt.Errorf("matchertest: record states no detail %q, want %+v", name, value)
		}
		if !matches(reflect.ValueOf(held), reflect.ValueOf(value)) {
			return fmt.Errorf("matchertest: detail %q is %+v, want %+v", name, held, value)
		}
	}
	return nil
}

// matches reports whether held, a value of a record's detail, matches
// value, the value that a case states for it, at any depth. An error
// matches a stated error under [errors.Is], a NaN matches a NaN, and a
// *big.Int matches a stated one of its value. Any other value matches as
// [reflect.DeepEqual] compares it. A case states no value that contains
// itself.
//
// The verdict of a case uses no comparison of internal/matcher, because a
// defective comparison could then pass the cases that check it.
func matches(held, value reflect.Value) bool {
	if !held.IsValid() || !value.IsValid() {
		return held.IsValid() == value.IsValid()
	}
	if held.CanInterface() && value.CanInterface() {
		if same, ok := matchesByMeaning(held.Interface(), value.Interface()); ok {
			return same
		}
	}
	if held.Type() != value.Type() {
		return false
	}
	switch held.Kind() {
	case reflect.Float32, reflect.Float64:
		a, b := held.Float(), value.Float()
		return a == b || math.IsNaN(a) && math.IsNaN(b)
	case reflect.Interface, reflect.Pointer:
		if held.IsNil() || value.IsNil() {
			return held.IsNil() && value.IsNil()
		}
		return matches(held.Elem(), value.Elem())
	case reflect.Slice:
		return held.IsNil() == value.IsNil() && elementsMatch(held, value)
	case reflect.Array:
		return elementsMatch(held, value)
	case reflect.Map:
		return held.IsNil() == value.IsNil() && entriesMatch(held, value)
	case reflect.Struct:
		for i := range held.NumField() {
			if !matches(held.Field(i), value.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Func:
		return held.IsNil() && value.IsNil()
	}
	return held.Equal(value)
}

// matchesByMeaning compares the two kinds of value that match by meaning
// and not by structure, and reports whether x and y are of them: two errors
// match under errors.Is in either direction, and two *big.Int match by
// value.
func matchesByMeaning(x, y any) (same, ok bool) {
	if a, isError := x.(error); isError {
		if b, isError := y.(error); isError {
			return errors.Is(a, b) || errors.Is(b, a), true
		}
	}
	a, aIsInt := x.(*big.Int)
	b, bIsInt := y.(*big.Int)
	if !aIsInt || !bIsInt {
		return false, false
	}
	if a == nil || b == nil {
		return a == b, true
	}
	return a.Cmp(b) == 0, true
}

// elementsMatch reports whether the arrays or slices held and value have
// the same length and matching elements.
func elementsMatch(held, value reflect.Value) bool {
	if held.Len() != value.Len() {
		return false
	}
	for i := range held.Len() {
		if !matches(held.Index(i), value.Index(i)) {
			return false
		}
	}
	return true
}

// entriesMatch reports whether the maps held and value have the same keys,
// and a matching value for each.
func entriesMatch(held, value reflect.Value) bool {
	if held.Len() != value.Len() {
		return false
	}
	for it := held.MapRange(); it.Next(); {
		v := value.MapIndex(it.Key())
		if !v.IsValid() || !matches(it.Value(), v) {
			return false
		}
	}
	return true
}

// checkOutcome fails t when the seat's outcome is not what the case
// required.
func checkOutcome(t *testing.T, seat *Seat, tc Case) {
	t.Helper()

	if err := Verdict(seat, tc); err != nil {
		t.Fatal(err)
	}
}
