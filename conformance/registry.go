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
// declares, which is what stops the table falling behind the definition
// or the surfaces.
//
// An assertion whose arguments are not expressible as typed literals
// has no entry: a callable, a context, a predicate or a golden file
// cannot cross a language boundary as data. A case can name a behaviour
// for some of them instead, and [SubjectDrivers] drives those.
var Registry = map[ID]map[Form]Invoker{
	"equal": {
		AbortingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			assert.Equal(tb, a[0], a[1], m, o...)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			expect.Equal(tb, a[0], a[1], m, o...)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, o []assert.Option) {
			assert.That(tb, a[0]).Equal(a[1], m, o...)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, o []assert.Option) {
			expect.That(tb, a[0]).Equal(a[1], m, o...)
		},
	},
	"not-equal": {
		AbortingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			assert.NotEqual(tb, a[0], a[1], m, o...)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			expect.NotEqual(tb, a[0], a[1], m, o...)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, o []assert.Option) {
			assert.That(tb, a[0]).NotEqual(a[1], m, o...)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, o []assert.Option) {
			expect.That(tb, a[0]).NotEqual(a[1], m, o...)
		},
	},
	"true": {
		AbortingCall:  func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.True(tb, a[0].(bool), m) },
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.True(tb, a[0].(bool), m) },
	},
	"false": {
		AbortingCall:  func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.False(tb, a[0].(bool), m) },
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.False(tb, a[0].(bool), m) },
	},
	"nil": {
		AbortingCall:   func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.Nil(tb, a[0], m) },
		RecordingCall:  func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.Nil(tb, a[0], m) },
		AbortingChain:  func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.That(tb, a[0]).Nil(m) },
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.That(tb, a[0]).Nil(m) },
	},
	"not-nil": {
		AbortingCall:   func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.NotNil(tb, a[0], m) },
		RecordingCall:  func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.NotNil(tb, a[0], m) },
		AbortingChain:  func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.That(tb, a[0]).NotNil(m) },
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.That(tb, a[0]).NotNil(m) },
	},
	"length": {
		AbortingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.Length(tb, a[0], a[1].(int), m)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.Length(tb, a[0], a[1].(int), m)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.That(tb, a[0]).Length(a[1].(int), m)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.That(tb, a[0]).Length(a[1].(int), m)
		},
	},
	"empty": {
		AbortingCall:   func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.Empty(tb, a[0], m) },
		RecordingCall:  func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.Empty(tb, a[0], m) },
		AbortingChain:  func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.That(tb, a[0]).Empty(m) },
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.That(tb, a[0]).Empty(m) },
	},
	"not-empty": {
		AbortingCall:   func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.NotEmpty(tb, a[0], m) },
		RecordingCall:  func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.NotEmpty(tb, a[0], m) },
		AbortingChain:  func(tb assert.TB, a []any, m string, _ []assert.Option) { assert.That(tb, a[0]).NotEmpty(m) },
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) { expect.That(tb, a[0]).NotEmpty(m) },
	},
	"contains": {
		AbortingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			assert.Contains(tb, a[0], a[1], m, o...)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			expect.Contains(tb, a[0], a[1], m, o...)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, o []assert.Option) {
			assert.That(tb, a[0]).Contains(a[1], m, o...)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, o []assert.Option) {
			expect.That(tb, a[0]).Contains(a[1], m, o...)
		},
	},
	"not-contains": {
		AbortingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			assert.NotContains(tb, a[0], a[1], m, o...)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			expect.NotContains(tb, a[0], a[1], m, o...)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, o []assert.Option) {
			assert.That(tb, a[0]).NotContains(a[1], m, o...)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, o []assert.Option) {
			expect.That(tb, a[0]).NotContains(a[1], m, o...)
		},
	},
	"contains-in-order": {
		AbortingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.ContainsInOrder(tb, a[0], a[1].([]string), m)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.ContainsInOrder(tb, a[0], a[1].([]string), m)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.That(tb, a[0]).ContainsInOrder(a[1].([]string), m)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.That(tb, a[0]).ContainsInOrder(a[1].([]string), m)
		},
	},
	"permutation": {
		AbortingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			permutation(tb, a, m, o, assert.Permutation[int], assert.Permutation[float64], assert.Permutation[any])
		},
		RecordingCall: func(tb assert.TB, a []any, m string, o []assert.Option) {
			permutation(tb, a, m, o, expect.Permutation[int], expect.Permutation[float64], expect.Permutation[any])
		},
	},
	"has-prefix": {
		AbortingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.HasPrefix(tb, a[0], a[1].(string), m)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.HasPrefix(tb, a[0], a[1].(string), m)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.That(tb, a[0]).HasPrefix(a[1].(string), m)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.That(tb, a[0]).HasPrefix(a[1].(string), m)
		},
	},
	"has-suffix": {
		AbortingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.HasSuffix(tb, a[0], a[1].(string), m)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.HasSuffix(tb, a[0], a[1].(string), m)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.That(tb, a[0]).HasSuffix(a[1].(string), m)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.That(tb, a[0]).HasSuffix(a[1].(string), m)
		},
	},
	"matches": {
		AbortingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.Matches(tb, a[0], a[1].(string), m)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.Matches(tb, a[0], a[1].(string), m)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.That(tb, a[0]).Matches(a[1].(string), m)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.That(tb, a[0]).Matches(a[1].(string), m)
		},
	},
	"close-to": {
		AbortingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.CloseTo(tb, a[0], a[1].(float64), a[2].(float64), m)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.CloseTo(tb, a[0], a[1].(float64), a[2].(float64), m)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.That(tb, a[0]).CloseTo(a[1].(float64), a[2].(float64), m)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.That(tb, a[0]).CloseTo(a[1].(float64), a[2].(float64), m)
		},
	},
	"in-range": {
		AbortingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.InRange(tb, a[0], a[1].(float64), a[2].(float64), m)
		},
		RecordingCall: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.InRange(tb, a[0], a[1].(float64), a[2].(float64), m)
		},
		AbortingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			assert.That(tb, a[0]).InRange(a[1].(float64), a[2].(float64), m)
		},
		RecordingChain: func(tb assert.TB, a []any, m string, _ []assert.Option) {
			expect.That(tb, a[0]).InRange(a[1].(float64), a[2].(float64), m)
		},
	},
}

// permutation calls the instance of a surface's Permutation that the
// element type of a case's decoded slices names: a slice of int, of
// float64, or of the literals of a list of items, which is a []any.
func permutation(tb assert.TB, a []any, m string, o []assert.Option,
	ints func(assert.TB, []int, []int, string, ...assert.Option),
	floats func(assert.TB, []float64, []float64, string, ...assert.Option),
	items func(assert.TB, []any, []any, string, ...assert.Option),
) {
	switch got := a[0].(type) {
	case []int:
		ints(tb, got, a[1].([]int), m, o...)
	case []float64:
		floats(tb, got, a[1].([]float64), m, o...)
	default:
		items(tb, a[0].([]any), a[1].([]any), m, o...)
	}
}

// Relaxations maps each relaxation that this language offers to its
// option, so a case's options become the options of its call.
var Relaxations = map[ID]assert.Option{
	"equate-empty": assert.EquateEmpty(),
	"equate-nans":  assert.EquateNaNs(),
}
