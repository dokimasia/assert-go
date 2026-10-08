// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"reflect"
	"slices"

	"go.dokimi.dev/assert/internal/equality"
)

// Spec is the sequential specification of an object: its state before any
// call, and the states that an operation may leave. S is the type of a
// state. [Linearizable] checks that the calls of a history have an order
// that respects their real-time order and that the spec accepts.
//
// Initial and Next are required. Equal and Hash are optional:
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
//     The first check of a history against such a spec notes the cause in
//     the log of its seat.
//
// Equal must be exact. The search merges two configurations whose states
// Equal reports equal, so an equality coarser than the spec's meaning turns
// a linearizable history into a violated one. A stated Hash agrees with
// Equal: two states that Equal reports equal have one hash. The search also
// takes two states of different hashes for different states.
//
// Next, Equal and Hash must not change their arguments. Under [Workers]
// they run on several goroutines at once, so they must be safe for that. A
// spec of pure functions is.
//
// The search copies the states that Next returns, and keeps no slice that
// Next returns. A spec may return one shared slice for a list of states
// that it leaves often, as long as nothing writes to that slice. A spec of
// states other than a basic type, which the search steps millions of times,
// is faster with an Equal and a Hash of its own than with the reflection of
// the defaults.
type Spec[S any] struct {
	// Initial returns the state before any call.
	Initial func() S
	// Next returns the states that may follow state when op takes effect, in
	// the order that the search tries them, and none when the spec rejects
	// op in state. For a call whose outcome is unknown, it returns the states
	// that the call leaves when it takes effect. It returns the same states
	// for the same arguments.
	Next func(state S, op Operation) []S
	// Equal reports whether two states are interchangeable.
	Equal func(a, b S) bool
	// Hash returns a hash of state that every state equal to it shares.
	Hash func(state S) uint64
	// cost returns the steps of the budget that a call of Next from state
	// counts for, and is nil for a spec whose every step counts for one.
	cost func(state S) int
}

// Operation is a call as a spec sees it: its operation and its arguments,
// and its output when the call completed as [OK].
type Operation struct {
	// Name is the call's operation.
	Name string
	// Args are the call's arguments.
	Args []any
	// Known reports whether the call completed as [OK]. A call whose outcome
	// is unknown, and a pending call, is not known.
	Known bool
	// Output is what the call returned when Known is set, and nil otherwise.
	Output any
}

// Returned reports whether the call may have returned v. A call that is not
// Known may have taken effect with any output, so Returned reports true for
// it. For a Known call, it reports whether Output equals v as assert.Equal
// compares them, without an option:
//
//	case "read":
//		if op.Returned(s) {
//			return []int{s}
//		}
//
// # Allocation contract
//
// Returned allocates nothing for an output of a basic type, such as an int
// or a string.
func (o Operation) Returned(v any) bool {
	return !o.Known || equality.Equal(reflect.ValueOf(o.Output), reflect.ValueOf(v), equality.Rules{})
}

// Subject applies an operation with its arguments to a subject, and returns
// the output.
type Subject func(operation string, args []any) any

// SpecFrom returns the spec that the subject's own sequential behaviour
// states. Its state is the list of the calls applied so far: the operation
// and the arguments of each, as an Operation without an output. A check
// against it finds a call of a history that was not atomic.
//
// Next builds a subject with factory, applies the calls of the state in
// order, and then applies the operation. It accepts the operation when the
// call may have returned the subject's output, as [Operation.Returned]
// reports. Two states are equal when they list equal operations with equal
// arguments, in one order. A step from a state of d calls applies d + 1
// calls, and counts for d + 1 steps of the budget.
//
// The subject must return the same outputs for the same calls. A subject
// with hidden state, such as a cache keyed by a pointer, returns other
// outputs on a replay, and the check then reports a violation that did not
// happen.
//
// # Allocation contract
//
// SpecFrom allocates the closure of the spec's Next.
func SpecFrom(factory func() Subject) Spec[[]Operation] {
	return Spec[[]Operation]{
		Initial: func() []Operation { return nil },
		Next: func(state []Operation, op Operation) [][]Operation {
			subject := factory()
			for _, applied := range state {
				subject(applied.Name, applied.Args)
			}
			if !op.Returned(subject(op.Name, op.Args)) {
				return nil
			}
			return [][]Operation{append(slices.Clip(state), Operation{Name: op.Name, Args: op.Args})}
		},
		cost: func(state []Operation) int { return len(state) + 1 },
	}
}
