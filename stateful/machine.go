// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import (
	"fmt"

	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/prop"
)

// Machine is the sequential specification of a subject and the actions that
// the steps of a case take on it. S is the type of the spec's state.
//
// The functions of a machine and of its actions receive a state of the
// spec: the first of the states that the last check of the history left. A
// machine without a spec receives the zero S.
//
// Those states follow what the subject returned, and a replay of a case
// requests the same choices only when its steps list the same actions and
// its requests state the same bounds. When a subject can return other
// results on a replay, as one that refuses a call for room or commits a
// batch in the background does, Enabled and Input read a state of the
// machine's own in place of the spec's: one that only the inputs of its
// steps change. Otherwise a replay can take other steps than the case took,
// and the run ends as flaky.
type Machine[S any] struct {
	// Spec is the sequential specification that the history of the steps is
	// checked against after every step, with every call in one partition. A
	// Spec that states neither Initial nor Next checks nothing.
	Spec history.Spec[S]
	// Actions are the actions of the machine, in order, the simpler first.
	// No two of them have one name.
	Actions []Action[S]
	// Invariant runs after setup, after every sequential and drain step, and
	// after Settle. It is optional.
	Invariant func(c *prop.Case, state S)
	// Settle runs once after the drain. It checks what must be true once the
	// subject has recovered, such as that every accepted write is readable.
	// It is optional.
	Settle func(c *prop.Case, state S)
}

// Action is one named action of a machine.
type Action[S any] struct {
	// Name names the action in a counterexample and in the step entries of a
	// trace.
	Name string
	// Weight is how often a random step takes the action, relative to the
	// other actions that the step lists. A Weight of 0 counts as 1, and a
	// negative Weight states no action.
	Weight int
	// Enabled reports whether a sequential or a drain step may take the
	// action in state. A nil Enabled enables the action in every state. Only
	// an action without Enabled takes the steps of a concurrent section.
	// What it reports must not depend on results that the subject can return
	// differently on a replay of the case.
	Enabled func(state S) bool
	// Drain marks an action that takes the steps of the drain, whether the
	// swarm kept it or not.
	Drain bool
	// Input requests the input of a step from the case, such as a draw. A nil
	// Input gives every step the input nil. The bounds of its requests must
	// not depend on results that the subject can return differently on a
	// replay of the case.
	Input func(c *prop.Case, state S) any
	// Run calls the subject on client with input, and records each call in
	// the case's history. It is required.
	Run func(c *prop.Case, client int, input any)
}

// validate panics for a machine that states no machine: an action with a
// negative weight or without Run, and two actions with one name.
func (m Machine[S]) validate() {
	names := make(map[string]struct{}, len(m.Actions))
	for _, a := range m.Actions {
		if a.Weight < 0 {
			panic(fmt.Sprintf("stateful: Steps of a machine whose action %q has weight %d", a.Name, a.Weight))
		}
		if a.Run == nil {
			panic(fmt.Sprintf("stateful: Steps of a machine whose action %q has no Run", a.Name))
		}
		if _, repeated := names[a.Name]; repeated {
			panic(fmt.Sprintf("stateful: Steps of a machine whose actions name %q twice", a.Name))
		}
		names[a.Name] = struct{}{}
	}
}

// specified reports whether the machine has a spec to check the history
// against.
func (m Machine[S]) specified() bool {
	return m.Spec.Initial != nil || m.Spec.Next != nil
}
