// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import "fmt"

// Strategy is how a [Scheduler] chooses the ready task to release. The zero
// Strategy is [Uniform].
type Strategy struct {
	// depth is the depth of PCT, and 0 for Uniform.
	depth int
}

// Uniform returns the strategy that chooses each release among the ready
// tasks, in the order that they became ready, uniformly, with target 0 and
// edge 0. A release with one ready task records a choice that consumes
// nothing. A shrunk schedule releases the tasks in the order that they
// became ready, wherever the failure allows.
func Uniform() Strategy {
	return Strategy{}
}

// PCT returns the strategy of probabilistic concurrency testing of depth
// depth. Each task receives a priority when it is spawned, a choice over
// [0, 2^64 - 1] with target 0, and keeps it when it yields. The ready task
// with the highest priority runs, the earliest ready among equals, and a
// release makes no choice. Each [Scheduler.Run] starts with depth - 1
// change points, each a presence choice with edge 1 and then a count of
// releases. The task released at that count falls below every other task,
// and a later change point puts its task lower still. It panics for a depth
// below 1.
func PCT(depth int) Strategy {
	if depth < 1 {
		panic(fmt.Sprintf("stateful: PCT(%d) is below 1", depth))
	}
	return Strategy{depth: depth}
}
