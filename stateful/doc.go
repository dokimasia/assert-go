// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package stateful runs a machine's steps inside a property's case: the
// sequential specification of a subject, actions over it, and the task
// scheduler that releases the tasks of a subject in an order that the case
// decides.
//
// A [Machine] lists its actions in order, the simpler first, and states the
// sequential specification of the subject. [Steps] takes the steps of a
// case, and records every decision as a choice of the case, so a failing
// case shrinks to the steps that the failure needs and replays from its
// token:
//
//	func TestQueue(t *testing.T) {
//		prop.ForAll(t, "the queue keeps its values in order", func(c *prop.Case) {
//			q := NewQueue(4)
//			written := 0
//			stateful.Steps(c, stateful.Machine[[]int]{
//				Spec: history.Spec[[]int]{
//					Initial: func() []int { return nil },
//					Next:    queueNext,
//				},
//				Actions: []stateful.Action[[]int]{{
//					Name:  "put",
//					Input: func(*prop.Case, []int) any { written++; return written },
//					Run: func(c *prop.Case, client int, v any) {
//						call := c.History().Invoke(client, "put", []any{v})
//						call.OK(q.Put(v.(int)))
//					},
//				}, {
//					Name: "get",
//					Run: func(c *prop.Case, client int, _ any) {
//						call := c.History().Invoke(client, "get", nil)
//						v, _ := q.Get()
//						call.OK(v)
//					},
//				}},
//			}, stateful.Clients(2))
//		})
//	}
//
// After every step, a machine with a spec checks the case's history with
// [history.Linearizable], with every call in one partition, and the next
// step reads the first of the states that the linearization it found
// leaves. Each check continues the search of the check before it through
// [history.Resume], so it searches the calls of the steps since that check.
// A counterexample lists each step as a [prop.Step] among the values that
// the case drew.
//
// # Concurrent sections
//
// With [Clients] of 2 or more, a case lists the steps of a concurrent
// section after its sequential steps and then runs them on the clients at
// once. On threads, the clients run through [history.Concurrently] and the
// case runs up to [Repeat] times, because threads do not replay. Under
// [Tasks], the clients run as tasks of a [Scheduler], whose releases are
// choices of the case, so a race replays and shrinks. A subject that runs
// as tasks starts its goroutines through [Scheduler.Spawn] and calls
// [Scheduler.Yield] where another task may run.
//
// # Traces
//
// The case of [prop.Draws] follows a trace of draw entries and step
// entries, such as a counterexample lists or a production incident states,
// and shrinks it when it fails.
//
// # Panics
//
// Steps panics for a machine that states no machine: an action with a
// negative weight or without Run, and two actions with one name. An option
// panics for a count below its minimum, [PCT] for a depth below 1, and
// [Tasks] for a nil scheduler. A scheduler panics when a task calls Run of
// its own scheduler. Each message starts with the package and names the
// call.
//
// # Concurrency
//
// Steps runs on the goroutine that runs the body, and makes every choice
// there. The functions of a machine run there too, except the actions of
// clients 1 and up of a concurrent section, which run on goroutines of
// their own: at once on threads, and one at a time as tasks.
//
// # Dependency position
//
// Imports the root package of this module, its prop and history packages,
// its internal engine, choice and random packages, and the standard
// library. No package of this module imports it except conformance.
package stateful
