// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/record"

// Observer is a [TB] that observes each assertion call, a passing one
// included. A seat of another library, such as the handle of a generated
// check, learns through it which assertions a check called.
//
// The first Observer of a seat's chain, which [TB] states, receives each call
// that a recorded run records, whatever DOKIMI_ASSERT_RECORD states: every
// verdict, and the calls in the body of a property, of [Eventually] and of
// [Rejects]. It receives each call once, before the seat receives the
// failure, the fault or the skip of that call. It receives the calls of a
// body when the call that ran the body takes them, so the calls of a
// property's cases arrive in the order of a run on one worker. It receives
// no call of [Fault], which is no assertion. ObserveCall may run on any
// goroutine of the test.
//
// An assertion on a chain without an Observer does not allocate for it. A
// property on a chain with an Observer keeps the calls of its cases, as a
// recorded property does.
type Observer = record.Observer

// Call is one assertion call that an [Observer] receives: the id of the
// assertion, as the definition names it, and the contract that the caller
// passed.
type Call = record.Observation
