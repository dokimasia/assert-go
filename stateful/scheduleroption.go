// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import "fmt"

// defaultMaxReleases is the most tasks that one Run or one Advance releases
// when no option states it, which the definition fixes.
const defaultMaxReleases = 1000000

// SchedulerOption configures a [Scheduler] that [NewScheduler] returns. The
// zero SchedulerOption keeps every default.
//
// # Allocation contract
//
// An option allocates nothing.
type SchedulerOption struct {
	// maxReleases is the most tasks that one Run or one Advance releases, and
	// 0 for the default.
	maxReleases int
}

// MaxReleases sets the most tasks that one [Scheduler.Run] or one
// [Scheduler.Advance] releases, 1,000,000 by default. A run or an advance
// that has released n tasks while a task is still ready ends every task and
// fails the case with a record without an assertion, whose contract is "the
// task scheduler released n tasks in one run, and a task is still ready".
// Under [Uniform] each release records a choice, so a case passes
// [go.dokimi.dev/assert/prop.MaxChoices] first unless the run raises it. It
// panics for n below 1.
func MaxReleases(n int) SchedulerOption {
	if n < 1 {
		panic(fmt.Sprintf("stateful: MaxReleases(%d) is below 1", n))
	}
	return SchedulerOption{maxReleases: n}
}
