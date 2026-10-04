// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"reflect"
	"slices"

	"go.dokimi.dev/assert/internal/equality"
)

// Model is a sequential model of a subject: the state before any call, and
// the states that a call may leave. S is the type of a state. [Linearizable]
// checks a history against a model.
//
// Init and Step are required. Equal and Hash are optional:
//
//   - A nil Equal compares two states as assert.Equal compares them: by
//     type and value, with -0 equal to +0, a NaN equal to nothing, and a nil
//     slice or map unequal to an empty one. It compares a state of a basic
//     type, such as int or string, with Go's ==, and a state of any other
//     type through reflection.
//   - A nil Hash with a nil Equal hashes a state to agree with that
//     comparison.
//   - A nil Hash with a stated Equal hashes every state alike. The memo of
//     the search then compares the states of every configuration of one set
//     of calls, which is slower by orders of magnitude on a long history.
//
// Equal must be exact. The search merges two configurations whose states
// Equal reports equal, so an equality coarser than the model's meaning turns
// a linearizable history into a violated one. A stated Hash agrees with
// Equal: two states that Equal reports equal have one hash. The search also
// takes two states of different hashes for different states.
//
// Step, Equal and Hash must not change their arguments. Under [Workers]
// they run on several goroutines at once, so they must be safe for that. A
// model of pure functions is.
//
// The search copies the states that Step returns, and keeps no slice that
// Step returns. A model may return one shared slice for a list of states
// that it leaves often, as long as nothing writes to that slice. A model of
// states other than a basic type, which the search steps millions of times,
// is faster with an Equal and a Hash of its own than with the reflection of
// the defaults.
type Model[S any] struct {
	// Init returns the state before any call.
	Init func() S
	// Step returns the states that may follow state when the call that op
	// describes takes effect, in the order that the search tries them, and
	// none when the model rejects the call in state. For a call whose outcome
	// is unknown, it returns the states that the call leaves when it takes
	// effect. It returns the same states for the same arguments.
	Step func(state S, op Op) []S
	// Equal reports whether two states are interchangeable.
	Equal func(a, b S) bool
	// Hash returns a hash of state that every state equal to it shares.
	Hash func(state S) uint64
	// cost returns the steps of the budget that a call of Step from state
	// counts for, and is nil for a model whose every step counts for one.
	cost func(state S) int
}

// Op is a call as a model sees it.
type Op struct {
	// Operation is the call's operation.
	Operation string
	// Args are the call's arguments.
	Args []any
	// Known reports whether the call completed as [OK]. A call whose outcome
	// is unknown, and a pending call, is not known.
	Known bool
	// Output is what the call returned when Known is set, and nil otherwise.
	Output any
}

// Subject applies an operation with its arguments to a subject, and returns
// the output.
type Subject func(operation string, args []any) any

// ModelFrom returns a model whose state is the list of the calls applied so
// far: the operation and the arguments of each, as an Op without an output.
// It checks that the calls of a history were atomic, with the subject's own
// sequential behaviour as the specification.
//
// Step builds a subject with factory, applies the calls of the state in
// order, and then applies the call. It accepts the call when the subject's
// output equals the recorded output, as assert.Equal compares them, and it
// accepts a call whose outcome is unknown. Two states are equal when they
// list equal operations with equal arguments, in one order. A step from a
// state of d calls applies d + 1 calls, and counts for d + 1 steps of the
// budget.
//
// The subject must return the same outputs for the same calls. A subject
// with hidden state, such as a cache keyed by a pointer, returns other
// outputs on a replay, and the check then reports a violation that did not
// happen.
//
// # Allocation contract
//
// ModelFrom allocates the closure of the model's Step.
func ModelFrom(factory func() Subject) Model[[]Op] {
	return Model[[]Op]{
		Init: func() []Op { return nil },
		Step: func(state []Op, op Op) [][]Op {
			subject := factory()
			for _, applied := range state {
				subject(applied.Operation, applied.Args)
			}
			output := subject(op.Operation, op.Args)
			if op.Known && !equality.Equal(reflect.ValueOf(output), reflect.ValueOf(op.Output), equality.Rules{}) {
				return nil
			}
			return [][]Op{append(slices.Clip(state), Op{Operation: op.Operation, Args: op.Args})}
		},
		cost: func(state []Op) int { return len(state) + 1 },
	}
}
