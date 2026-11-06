---
rfc: 0026
title: Definition 16.0.0 in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-11-05
updated: 2026-11-05
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0026: Definition 16.0.0 in Go

## Summary

The module vendors definition 16.0.0. A task of the scheduler can park on a condition or on a deadline of the scheduler's own clock, and a run of a scheduler has a limit of releases. A run of a property replays a failing case a stated number of times.

| Change | Go API |
|---|---|
| A task parks until a condition is true | `Scheduler.Wait` |
| A task sleeps on the clock of its scheduler, and the body moves that clock from deadline to deadline | `Scheduler.Clock`, `Scheduler.Advance` |
| A run or an advance fails its case once it has released its limit of tasks while a task is still ready | `stateful.MaxReleases`, `stateful.SchedulerOption`, the variadic options of `NewScheduler` |
| A run replays a failing case n times before the shrink, and counts the replays that reproduce the failure | `prop.Replays`, `Result.Replays`, `Result.Reproduced` |

Every call record states `16.0.0`.

## Motivation

The definition's RFC-0056 and RFC-0057 argue the changes. This RFC fixes the Go design of the changes:

- the public names that the naming table does not state
- how a parked task waits
- how the conformance runner identifies the failure of the limit

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| The parked tasks | `stateful/scheduler.go` | `Wait`, `Advance`, the waiting and the sleeping tasks, and their end with the case |
| The clock of a scheduler | `stateful/clock.go` | `Scheduler.Clock` and the unexported type of the clock |
| The options of a scheduler | `stateful/scheduleroption.go` | `SchedulerOption` and `MaxReleases` |
| The replays of a failing case | `internal/prop/engine/runner.go` | `Settings.Replays`, `Result.Replays` and `Result.Reproduced` |
| The option and the result | `prop/option.go`, `prop/result.go`, `prop/detail.go` | `Replays`, the two members of `Result`, and the two fields of the record |
| The runners | `conformance/machines.go`, `conformance/body.go`, `conformance/behaviour.go`, `conformance/record.go` | The five machine subjects, the setup key `max-releases`, the body `fails-on-even-calls`, the setting `replays` and the two fields |

### The parked tasks

```go
// Wait parks the running task until cond returns true. It returns at once
// when cond returns true at the call. Outside a task, on the goroutine that
// runs the body, it returns at once when cond returns true.
//
// # Panics
//
// Wait panics outside a task when cond returns false.
func (s *Scheduler) Wait(cond func() bool)

// Advance releases the ready tasks as Run does, and moves the clock to each
// deadline of a sleeping task up to d from now, in order. Advance returns
// once the clock reads its end time and no task is ready.
//
// # Panics
//
// Advance panics when a task calls it on its own scheduler.
func (s *Scheduler) Advance(d time.Duration)
```

- A task parks as it yields: it sends the end of its turn and waits on its channel for the next release. The turn states that the task parked, and the scheduler keeps the task as waiting or as sleeping.
- The scheduler keeps the waiting tasks in the order of their waits. It keeps the sleeping tasks sorted by deadline, with a later sleep after an earlier sleep of one deadline, so the due tasks are a prefix of the list.
- `Run` and `Advance` wake the parked tasks at their start, after each turn, and after each move of the clock. A condition runs on the goroutine that calls `Run` or `Advance`, which is the goroutine of the body. A panic of a condition passes through the call, as the panic of a task does.
- The end of a run and the cleanup of the case end the parked tasks as they end the ready tasks: each receives false on its channel and ends its goroutine through `runtime.Goexit`.

### The clock of a scheduler

```go
// Clock returns the clock of the scheduler. The clock starts at midnight UTC
// 2000-01-01, and only Advance moves it. Its Sleep parks the running task
// until the clock has passed the duration, and returns at once for a duration
// of zero or less.
//
// # Panics
//
// The Sleep of the clock panics outside a task for a positive duration.
func (s *Scheduler) Clock() assert.Clock
```

The clock is a value of an unexported struct type with one pointer to its scheduler, so the interface stores the value without an allocation. `Now` returns an instant in UTC.

### The options of a scheduler

```go
// SchedulerOption configures a Scheduler that NewScheduler returns. The zero
// SchedulerOption keeps every default.
type SchedulerOption struct{ /* unexported */ }

// MaxReleases sets the most tasks that one Run or one Advance releases,
// 1,000,000 by default. A run or an advance that has released n tasks while
// a task is still ready ends every task and fails the case. It panics for n
// below 1.
func MaxReleases(n int) SchedulerOption

func NewScheduler(c *prop.Case, strategy Strategy, opts ...SchedulerOption) *Scheduler
```

At the limit, the scheduler ends every task and calls `Fatalf` of the case with the message of the definition. The record has no assertion, and its contract is the message. Its location is the innermost frame of the caller's code. That frame is the call of `Run` or `Advance`.

| Name | Why Go needs it |
|---|---|
| `stateful.SchedulerOption` | `MaxReleases` returns a value that `NewScheduler` takes, as `stateful.Option` is the value that `Steps` takes |

### The replays of a failing case

```go
// Replays makes a run replay a failing case n times before it shrinks the
// case, from its choices. The run shrinks the case when one or more replays
// reproduce the failure, and ends as Flaky when none does. It panics for n
// below 1.
func Replays(n int) Option
```

- `engine.Settings.Replays` is the number of replays, and a run replays once for 0. `conclude` runs the replays one after the other through `confirm`, and every replay runs. A replay that ends with a fault ends the run with that fault.
- `engine.Result.Replays` and `Result.Reproduced` count the replays and the replays that reproduced the failure. `conclude` serves `Run`, `Conclude` and a campaign, so the fuzz bridge and a campaign replay as a run does.
- `prop.Result` gains `Replays` and `Reproduced`, two ints. A run that replays a failing case states at least 1 replay, so 0 stands for a run that replayed none. The record of a run states null for both fields then.

### The conformance runners

- The machines runner builds the five subjects of the definition in Go and reads the setup key `max-releases`. It gives the identity `max-releases` to a record without an assertion, because the limit of a scheduler is the only failure without an assertion that a subject of the definition reports. The vectors identify that failure by the same name.
- The machines runner compares the fault that ends a run, and not a fault that the run notes. The store refuses the entry of a failure without a location, and every frame of a subject of the runner is in the module, so the run of `spins-until-a-flag` notes that refusal.
- The behaviour runner reads the setting `replays`, and the bodies gain `fails-on-even-calls`.
- The detail of a run gains `replays` and `reproduced`, compared in the order of the definition.
- The completeness gate in `conformance/surface_test.go` pins the seven new names of the table.

### The record

`go generate` writes `16.0.0` from the vendored `VERSION` into `record.Definition`.

### Changesets

```markdown
Add `Scheduler.Wait`, `Scheduler.Clock` and `Scheduler.Advance` to `stateful` to park a task on a condition or on a deadline of the clock of its scheduler.
```

```markdown
**Breaking:** Fail a case whose `Scheduler.Run` or `Scheduler.Advance` releases `stateful.MaxReleases` tasks, 1,000,000 by default, while a task is still ready.
```

```markdown
**Breaking:** Add `prop.Replays` and the fields `replays` and `reproduced` of the detail of a run, as `Result.Replays` and `Result.Reproduced`.
```

```markdown
Implement definition 16.0.0 and write its version into every call record.
```

## Alternatives considered

### A. The limit as an option of Steps

`MaxReleases` would return a `stateful.Option`, and `NewScheduler` would take the options of `Steps`.

**Why not:** a scheduler would then accept every option of `Steps` and ignore all but one. `Steps` would accept the limit and ignore it. With two types, the compiler refuses an option at a call that ignores it.

### B. An exported type of the clock

The package would export the type of the scheduler's clock.

**Why not:** a subject takes `assert.Clock`, which is the clock of the definition. The exported type would add a name that the table does not state, and it would have no method that the interface lacks.

### C. The replays at once on the workers

A run on more workers would run the replays of a failing case at once.

**Why not:** running the replays at once saves the wall time of at most n − 1 runs, and the shrink after them has a budget of 2,000 runs. The call records keep the order of a run on one worker in both designs.

## Drawbacks

- `NewScheduler` gains a variadic parameter. Every call compiles unchanged, and a variable of the old function type does not take it.
- Each turn ends with a check of the parked tasks: a call of the condition of each waiting task and a look at the first sleeping task.
- The machines runner gives every record without an assertion the identity `max-releases`. A subject of a later definition that fails without an assertion needs an identity of its own in the runner.
- A run that released more than 1,000,000 tasks in one call fails.

## Rules of the standards that this amends

The standard of errors and output gains three panics of a scheduler: `Wait` outside a task with a condition that is false, the `Sleep` of its clock outside a task for a positive duration, and `Advance` from a task of its own scheduler.

## References

| What | Where |
|---|---|
| Tasks that wait and sleep under the task scheduler | <https://github.com/dokimasia/assert-spec/blob/main/docs/rfc/0056-tasks-that-wait-and-sleep.md> |
| The replays of a failing case | <https://github.com/dokimasia/assert-spec/blob/main/docs/rfc/0057-the-replays-of-a-failing-case.md> |
| Machines, the task scheduler and campaigns in Go | RFC-0011 |
