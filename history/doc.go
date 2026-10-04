// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package history records the calls that concurrent clients make to a
// subject, and checks that the recorded history is linearizable with
// respect to a sequential model.
//
// A test records each call through a [History]: [History.Invoke] before the
// call to the subject starts, and [Call.OK], [Call.Fail] or [Call.Unknown]
// after it returns. [Concurrently] runs the clients, and [Linearizable]
// checks the history:
//
//	func TestRegisterIsLinearizable(t *testing.T) {
//		h := history.New()
//		reg := NewRegister()
//		history.Concurrently(2, time.Minute, func(client int) (any, error) {
//			for i := range 50 {
//				c := h.Invoke(client, "write", []any{client*1000 + i}, "x")
//				reg.Write(client*1000 + i)
//				c.OK(nil)
//				c = h.Invoke(client, "read", nil, "x")
//				c.OK(reg.Read())
//			}
//			return nil, nil
//		})
//		history.Linearizable(t, h, history.Model[int]{
//			Init: func() int { return 0 },
//			Step: func(s int, op history.Op) []int {
//				if op.Operation == "write" {
//					return []int{op.Args[0].(int)}
//				}
//				if !op.Known || op.Output == s {
//					return []int{s}
//				}
//				return nil
//			},
//		}, "the register is linearizable")
//	}
//
// The definition fixes the recording order, the processes, the partitions,
// the search, its budget and its record, so one history gives the same
// verdict and the same record in every implementation of the definition.
//
// # Histories
//
// One mutex orders every invocation and completion, so the events are in one
// order that every client observes. Call a precedes call b when a's
// completion comes before b's invocation. Each client starts on a process of
// its own, and a call whose outcome is unknown moves its client to a new
// process. The keys of an invocation name what the call touches, as typed
// literals, and a call without keys touches every key. [FromIntervals]
// builds a history from calls that a log recorded with a start and an end on
// one clock.
//
// # Checks
//
// [Linearizable] removes the calls that failed, partitions the history by
// its keys, and searches each partition for an order of its calls that keeps
// the precedence and that the model accepts. A [Model] states the state
// before any call and the states that a call may leave. [ModelFrom] builds a
// model from the subject itself, which checks that the calls were atomic. The
// search spends at most 10,000,000 steps and a memo of 1 GiB on each
// partition. A search that uses up either is [Undecided], which fails the
// test as [Violated] does. [Budget], [MemoLimit], [TimeLimit] and [Workers]
// change the limits and the number of partitions searched at once.
//
// # Panics
//
// A call that breaks the contract of the history panics, and the message
// names the call: an invocation by a client whose call is open, a second
// completion of one call, a key that no typed literal states, and
// Concurrently without clients or with a negative time. The constructors of
// the options panic for a value outside their domain.
//
// # Concurrency
//
// Every method of a [History] and of each [Call] is safe for concurrent use.
// Under [Workers], the functions of a model run on several goroutines at
// once.
//
// # Dependency position
//
// Imports the root package of this module, its internal equality, fault,
// literal, matcher and text packages, and the standard library. The
// conformance package imports it.
package history
