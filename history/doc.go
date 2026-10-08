// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package history records the calls that concurrent clients make to an
// object, and checks the recorded history: that it is linearizable, that its
// transactions are serializable, or that they have snapshot isolation. A
// history is linearizable when its calls have an order that respects their
// real-time order and that the object's sequential specification accepts.
//
// A test records each call through a [History]: [History.Invoke] before the
// call to the subject starts, and [Call.OK], [Call.Fail] or [Call.Unknown]
// after it returns. [Concurrently] runs the clients, and [Linearizable]
// checks the history against a [Spec]:
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
//		history.Linearizable(t, h, history.Spec[int]{
//			Initial: func() int { return 0 },
//			Next: func(s int, op history.Operation) []int {
//				switch op.Name {
//				case "write":
//					return []int{op.Args[0].(int)}
//				case "read":
//					if op.Returned(s) {
//						return []int{s}
//					}
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
// order that every client observes. Call a precedes call b in real time when
// a's completion comes before b's invocation. Each client starts on a process
// of its own, and a call whose outcome is unknown moves its client to a new
// process. The keys of an invocation name what the call touches, as typed
// literals, and a call without keys touches every key. [FromIntervals]
// builds a history from calls that a log recorded with a start and an end on
// one clock.
//
// # Checks
//
// [Linearizable] removes the calls that failed, partitions the history by
// its keys, and searches each partition for a linearization: an order of its
// calls that respects their real-time order and that the spec accepts. A
// [Spec] states the state before any call and the states that an
// [Operation] may leave. [SpecFrom] builds the spec from the subject itself,
// which checks that the calls were atomic. The search spends at most
// 10,000,000 steps and a memo of 1 GiB on each partition. A search that uses
// up either is [Undecided], which fails the test as [Violated] does.
// [Budget], [MemoLimit], [TimeLimit] and [Workers] change the limits and the
// number of partitions searched at once. [Whole] searches every call as one
// partition, and [Resume] continues the search of the last passing check of
// a history that has grown since, with the result of a search of every call.
//
// [Serializable] and [HasSnapshotIsolation] read each call of the operation
// "txn" as a transaction of list appends and list reads. They derive the
// dependencies between the committed transactions that the reads reveal,
// and search them for the anomalies that the isolation level forbids. Both
// always decide. A record names the first anomaly, every kind that the
// history exhibits, the transactions involved, the cycle, and the evidence
// of each of its dependencies.
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
// Under [Workers], the functions of a spec run on several goroutines at
// once.
//
// # Dependency position
//
// Imports the root package of this module, its internal equality, fault,
// literal, matcher and text packages, and the standard library. The
// property engine and the conformance package import it.
package history
