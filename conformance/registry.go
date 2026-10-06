// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

// Form is one way a caller states an assertion: a function of the
// aborting surface or of the recording one, or a method of either
// surface's chain.
type Form string

// The forms the standard requires. Every assertion has both function
// forms, and an assertion whose first argument is the value examined has
// both chain forms too.
const (
	// AbortingCall is a function of the aborting surface, assert.Equal.
	AbortingCall Form = "assert"
	// RecordingCall is a function of the recording surface, expect.Equal.
	RecordingCall Form = "expect"
	// AbortingChain is a method of the aborting chain, assert.That(...).Equal.
	AbortingChain Form = "assert.That"
	// RecordingChain is a method of the recording chain, expect.That(...).Equal.
	RecordingChain Form = "expect.That"
)

// Invoker drives one assertion in one form against a case's decoded
// arguments, its message and its relaxations.
type Invoker func(tb assert.TB, args []any, msg string, opts []assert.Option)

// Registry maps an assertion to the call that drives it in each form.
//
// Go cannot call a function by name at run time, so the corpus
// dispatches through this table. The corpus test requires an entry in
// both function forms for every assertion that a case states values
// for, and in both chain forms for every one whose name the chain
// declares. A table that lacks an assertion of the definition or a method
// of a chain fails that test.
//
// An assertion whose arguments are not expressible as typed literals
// has no entry: a callable, a context, a predicate or a golden file
// cannot cross a language boundary as data. A case can name a behaviour
// for some of them instead, and [SubjectDrivers] drives those.
var Registry = registry(map[Form]map[ID]Invoker{
	AbortingCall:   callInvokers(abortingFunctions),
	RecordingCall:  callInvokers(recordingFunctions),
	AbortingChain:  chainInvokers(assert.That[any]),
	RecordingChain: chainInvokers(expect.That[any]),
})

// registry returns the invokers of each form, keyed by the assertion and
// then by the form.
func registry(forms map[Form]map[ID]Invoker) map[ID]map[Form]Invoker {
	out := make(map[ID]map[Form]Invoker)
	for form, invokers := range forms {
		for id, invoke := range invokers {
			if out[id] == nil {
				out[id] = make(map[Form]Invoker)
			}
			out[id][form] = invoke
		}
	}
	return out
}

// callInvokers returns the invoker of the function form of each assertion
// of the surface whose functions are f.
func callInvokers(f functions) map[ID]Invoker {
	return map[ID]Invoker{
		"equal": func(tb assert.TB, a []any, m string, o []assert.Option) { f.Equal(tb, a[0], a[1], m, o...) },
		"not-equal": func(tb assert.TB, a []any, m string, o []assert.Option) {
			f.NotEqual(tb, a[0], a[1], m, o...)
		},
		"true":      func(tb assert.TB, a []any, m string, _ []assert.Option) { f.True(tb, a[0].(bool), m) },
		"false":     func(tb assert.TB, a []any, m string, _ []assert.Option) { f.False(tb, a[0].(bool), m) },
		"nil":       func(tb assert.TB, a []any, m string, _ []assert.Option) { f.Nil(tb, a[0], m) },
		"not-nil":   func(tb assert.TB, a []any, m string, _ []assert.Option) { f.NotNil(tb, a[0], m) },
		"length":    func(tb assert.TB, a []any, m string, _ []assert.Option) { f.Length(tb, a[0], a[1].(int), m) },
		"empty":     func(tb assert.TB, a []any, m string, _ []assert.Option) { f.Empty(tb, a[0], m) },
		"not-empty": func(tb assert.TB, a []any, m string, _ []assert.Option) { f.NotEmpty(tb, a[0], m) },
		"contains": func(tb assert.TB, a []any, m string, o []assert.Option) {
			f.Contains(tb, a[0], a[1], m, o...)
		},
		"not-contains": func(tb assert.TB, a []any, m string, o []assert.Option) {
			f.NotContains(tb, a[0], a[1], m, o...)
		},
		"contains-in-order": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			f.ContainsInOrder(tb, a[0], a[1].([]string), m)
		},
		"permutation": func(tb assert.TB, a []any, m string, o []assert.Option) { f.permutation(tb, a, m, o) },
		"has-prefix": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			f.HasPrefix(tb, a[0], a[1].(string), m)
		},
		"has-suffix": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			f.HasSuffix(tb, a[0], a[1].(string), m)
		},
		"matches": func(tb assert.TB, a []any, m string, _ []assert.Option) { f.Matches(tb, a[0], a[1].(string), m) },
		"close-to": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			f.CloseTo(tb, a[0], a[1].(float64), a[2].(float64), m)
		},
		"in-range": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			f.InRange(tb, a[0], a[1].(float64), a[2].(float64), m)
		},
	}
}

// permutation calls the instance of the surface's Permutation that the
// element type of a case's decoded slices names: a slice of int, of
// float64, or of the literals of a list of items, which is a []any.
func (f functions) permutation(tb assert.TB, a []any, m string, o []assert.Option) {
	switch got := a[0].(type) {
	case []int:
		f.PermutationOfInts(tb, got, a[1].([]int), m, o...)
	case []float64:
		f.PermutationOfFloats(tb, got, a[1].([]float64), m, o...)
	default:
		f.PermutationOfItems(tb, a[0].([]any), a[1].([]any), m, o...)
	}
}

// chain is the chain of a surface, of the value examined as an any. Each
// method returns the chain, so C is the chain's own type.
type chain[C any] interface {
	Equal(want any, msg string, opts ...assert.Option) C
	NotEqual(want any, msg string, opts ...assert.Option) C
	Nil(msg string) C
	NotNil(msg string) C
	Length(want int, msg string) C
	Empty(msg string) C
	NotEmpty(msg string) C
	Contains(needle any, msg string, opts ...assert.Option) C
	NotContains(needle any, msg string, opts ...assert.Option) C
	ContainsInOrder(needles []string, msg string) C
	HasPrefix(prefix, msg string) C
	HasSuffix(suffix, msg string) C
	Matches(pattern, msg string) C
	CloseTo(want, tolerance float64, msg string) C
	InRange(low, high float64, msg string) C
}

// chainInvokers returns the invoker of the chain form of each assertion
// that the chain declares, over the chains that the surface's function
// That starts.
func chainInvokers[C chain[C]](that func(tb assert.TB, got any) C) map[ID]Invoker {
	return map[ID]Invoker{
		"equal": func(tb assert.TB, a []any, m string, o []assert.Option) { that(tb, a[0]).Equal(a[1], m, o...) },
		"not-equal": func(tb assert.TB, a []any, m string, o []assert.Option) {
			that(tb, a[0]).NotEqual(a[1], m, o...)
		},
		"nil":       func(tb assert.TB, a []any, m string, _ []assert.Option) { that(tb, a[0]).Nil(m) },
		"not-nil":   func(tb assert.TB, a []any, m string, _ []assert.Option) { that(tb, a[0]).NotNil(m) },
		"length":    func(tb assert.TB, a []any, m string, _ []assert.Option) { that(tb, a[0]).Length(a[1].(int), m) },
		"empty":     func(tb assert.TB, a []any, m string, _ []assert.Option) { that(tb, a[0]).Empty(m) },
		"not-empty": func(tb assert.TB, a []any, m string, _ []assert.Option) { that(tb, a[0]).NotEmpty(m) },
		"contains": func(tb assert.TB, a []any, m string, o []assert.Option) {
			that(tb, a[0]).Contains(a[1], m, o...)
		},
		"not-contains": func(tb assert.TB, a []any, m string, o []assert.Option) {
			that(tb, a[0]).NotContains(a[1], m, o...)
		},
		"contains-in-order": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			that(tb, a[0]).ContainsInOrder(a[1].([]string), m)
		},
		"has-prefix": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			that(tb, a[0]).HasPrefix(a[1].(string), m)
		},
		"has-suffix": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			that(tb, a[0]).HasSuffix(a[1].(string), m)
		},
		"matches": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			that(tb, a[0]).Matches(a[1].(string), m)
		},
		"close-to": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			that(tb, a[0]).CloseTo(a[1].(float64), a[2].(float64), m)
		},
		"in-range": func(tb assert.TB, a []any, m string, _ []assert.Option) {
			that(tb, a[0]).InRange(a[1].(float64), a[2].(float64), m)
		},
	}
}

// Relaxations maps each relaxation that this language offers to its
// option, so a case's options become the options of its call.
var Relaxations = map[ID]assert.Option{
	"equate-empty": assert.EquateEmpty(),
	"equate-nans":  assert.EquateNaNs(),
	"by-identity":  assert.ByIdentity(),
}
