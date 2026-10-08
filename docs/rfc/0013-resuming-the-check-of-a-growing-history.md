---
rfc: 0013
title: Resuming the check of a growing history
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-06
updated: 2026-10-06
discussion: https://github.com/dokimasia/assert-go/issues/17
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0013: Resuming the check of a growing history

## Summary

`Steps` checks the case's history after every sequential and drain step.
Each check runs `Linearizable` over the whole history, so the check after
step n searches all n calls again. A case of n steps then costs about
n²/2 model steps, and its memory grows with the square of n.

This proposal adds a type and an option to `history`:

```go
// Checkpoint keeps the search of the last check of a whole history that
// passed with it, so that the next check of the history continues that
// search instead of searching from the first call.
type Checkpoint[S any] struct{ /* unexported */ }

// Resume makes the check continue the search that cp keeps, and keep its
// own search in cp when it passes.
func Resume[S any](cp *Checkpoint[S]) Option
```

A check with `Resume` continues the kept search over the calls that the
history recorded since. The continued search takes the steps that a search
of every call takes after the order that the kept search found. The check
then reports what a search of the whole history reports: its verdict, its
steps, its frontier and its final states. `Steps` passes `Resume` to every
check. A case of at most 400 steps of a register machine takes 0.6 ms and
1.56 MB, against 15 ms and 78.5 MB.

## Motivation

### The checks of a case search every call again

A machine of `set` and `get` over a register behind a mutex shows the
growth. Each case runs up to the stated number of sequential steps, 20
cases at seed 1, measured under Go 1.27.1 at a `GOMAXPROCS` of 4:

| Steps at most | Steps a case | Time a case | Bytes a case | Allocations a case |
|---|---|---|---|---|
| 50 | 37.0 | 329 to 385 µs | 1.29 MB | 3,429 |
| 100 | 74.5 | 0.93 to 1.20 ms | 4.98 MB | 9,495 |
| 200 | 144.1 | 3.73 to 3.83 ms | 19.14 MB | 27,110 |
| 400 | 294.3 | 14.7 to 15.1 ms | 78.5 MB | 86,953 |

Each doubling of the steps multiplies the bytes of a case by 3.8 to 4.1. A
machine whose cases run hundreds of steps over three clients spends most
of its memory on the checks. A mutation run of its package then runs few
workers.

### The states that a machine reads

`Steps` reads the states of the first order that a check finds: `Enabled`
filters the next step's actions on them, and `Input` takes the first. A
check that starts from a cut of the history finds its orders in another
sequence than a search of the whole history does. The definition would
then need a rule for those states. Its vectors of machines would change
with the rule. A check that continues the kept search finds the order of
a search of the whole history, and the states and the vectors keep their
values.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| `Checkpoint` | `history/checkpoint.go` | Keeps the search of the last passing check, its history and the number of events that it read |
| `Resume` | `history/option.go` | The option that passes a checkpoint to a check |
| The grown search | `history/search.go` | Adds calls to a search that passed, and continues it |
| The check after a step | `stateful/steps.go` | Passes `Whole`, `Final` and `Resume` of the run's checkpoint |

### The equivalence of the continued search and a search of every call

The search scans the entries of the calls in event order. It takes the
first entry that it can linearize, and backtracks at the completion of a
known call that it has not linearized. It meets a new call's invocation
only after it has linearized every known call before that invocation. The
new calls' events follow every event that the kept search read. A search
of every call takes the steps of the kept search until the first order
that linearizes every known call of the kept history. The kept search
stopped there.

From that order on, the search of every call continues over the new
calls, and a backtrack can return into the earlier calls. The grown
search does the same from the same stack, memo, frontier and step count.
Its result is the result of the search of every call:

- the verdict, and the steps up to it
- the frontier's order of calls, its states and its candidates
- the limit that stopped an undecided search
- the final states of a passing check

### The grown search

Five changes let a search grow without changing a search that does not:

- **The head and the tail of the scan keep the indices 0 and 1.** A new
  call's entries take the next indices, so no index of an earlier entry
  changes.
- **A completion that the search took out while the tail followed it
  points at the first new entry.** A search of every call took it out
  while that entry followed it, so a backtrack puts it back before the new
  entries. No invocation that the search took out was the last entry: the
  scan takes an invocation out only while a known call is not linearized,
  and the completion of that call follows the invocation.
- **The set of linearized calls gains a word for each 64 new calls,** in
  the search and in each configuration of the memo.
- **The key of a configuration mixes the words of its set up to its last
  word with a call.** A configuration that the kept search stored keeps
  its key when the set gains words.
- **The capacity of the memo is the memo limit divided by every call.**

### The conditions for a search of every call

A check with `Resume` searches the whole history again, and keeps the new
search, when the kept search cannot continue as a search of every call:

| Condition | Why |
|---|---|
| The checkpoint keeps no search | The checkpoint is new, or the last check did not pass |
| The checkpoint keeps the search of another history | The calls differ |
| The budget or the memo limit differs | A search under other limits stops elsewhere |
| A new event completes a call that the kept search read as pending | The call changes: it becomes known, unknown, or leaves the history |
| The memo contains more configurations than the grown search may store | A search of every call uses up its memo limit before the order that the kept search found |

A check that does not pass, and a check whose model panics, keeps no
search in the checkpoint. A check with a time limit reads the clock before
the grown search's next step.

### The public names

```go
// Checkpoint keeps the search of the last check of a whole history that
// passed with it, so that the next check of the history continues that
// search instead of searching from the first call. [Resume] passes it to a
// check. The zero Checkpoint keeps nothing.
//
// # Concurrency
//
// A Checkpoint is safe for one check at a time.
type Checkpoint[S any] struct{ /* unexported */ }

// Resume makes the check continue the search that cp keeps, and keep its own
// search in cp when it passes. A check that does not pass keeps no search in
// cp. A nil cp changes nothing.
//
// Pass the same model to every check that passes cp. A check with Resume
// states [Whole], and its model has states of the type S. A check that
// breaks either rule ends the call with a fault.
func Resume[S any](cp *Checkpoint[S]) Option
```

`Resume` and `Checkpoint` are Go names that the naming table does not
state, as `Whole` and `Final` are. The definition fixes the result of a
check, which they do not change.

`Steps` keeps a `Checkpoint` in its run, and passes it to every check:

```go
history.Linearizable(&seat, c.History(), m.Model, contract,
	history.Whole(), history.Final(&states), history.Resume(&checkpoint))
```

### Measurements

The machine of the motivation, 20 cases at seed 1, in three alternating
rounds of the commit before this change and of this change, under Go
1.27.1 at a `GOMAXPROCS` of 4:

| Steps at most | Time a case, before | Time a case, after | Bytes a case, before | Bytes a case, after | Allocations a case, before | Allocations a case, after |
|---|---|---|---|---|---|---|
| 50 | 329 to 385 µs | 80 to 119 µs | 1.29 MB | 0.16 MB | 3,429 | 619 |
| 100 | 0.93 to 1.20 ms | 150 to 167 µs | 4.98 MB | 0.37 MB | 9,495 | 1,142 |
| 200 | 3.73 to 3.83 ms | 313 to 353 µs | 19.14 MB | 0.72 MB | 27,110 | 2,096 |
| 400 | 14.7 to 15.1 ms | 580 to 669 µs | 78.5 MB | 1.56 MB | 86,953 | 4,138 |

The bytes of a case now grow linearly with its steps. The ceiling of the
`stateful` benchmark, a case of 100 sequential steps of a queue that checks
the history 101 times, falls from 15,272 allocations to 1,437. A write that
a history records and the check that continues the search before it
allocate 10 times together.

### Testing

- A property of 300 cases records random histories of up to 30 events
  over three clients, with calls that complete as OK, Fail and Unknown and
  calls that remain pending, over a register and a register whose write may
  be lost. After each event it checks the history with `Resume` and
  without, and requires the same records and the same final states.
- A test for each condition of the table counts the model's steps and
  requires a search of every call.
- A test continues a grown search that backtracks into the calls before
  the growth, and one that finds a configuration that the kept search
  stored before the set of calls gained a word.
- The checks after each of 20 sequential calls step each call once.

## Alternatives considered

### A. A cut that starts from every configuration of the earlier check

The check would keep every configuration at the end of its search. Each
is a set of linearized calls and its states. The next check would start
from all the configurations with the calls recorded since.

**Why not:** a search stops at the first order that passes. Keeping every
configuration at the end costs a search of every order of each concurrent
section. The continued search runs such a search only when a later call
needs an order other than the first. The next check would also find its
orders in another sequence than a search of the whole history. The states
that `Steps` reads would change with them.

### B. The search in an internal package that `stateful` calls

`stateful` would keep a search of an internal package between its checks.
`history` would declare no new name.

**Why not:** the search reads the public `Op`, and the cost that
`ModelFrom` sets in an unexported field of `Model`. An internal package
would declare its own op and its own model. `history` would copy each
model into that form on every check.

### C. A search that the history keeps

`History` would keep the search of its last check of the whole history. A
check would continue it without an option.

**Why not:** the history records events, and a search of one model is not
part of them. Two checks of one history against two models would continue
each other's search without a way to tell them apart.

## Drawbacks

- **A checkpoint keeps its memo between checks:** the memory of a search
  of the whole history, released with the checkpoint.
- **A caller who passes another model with one checkpoint continues a
  search of the earlier model's states.** The doc comment requires one
  model for each checkpoint, and nothing checks it.
- **A check under a time limit can decide where a search of every call
  would run out of time,** because the grown search does not repeat the
  steps before the growth.
- **A check with `Resume` of an empty history calls `Init`,** which a check
  without `Final` or `Resume` does not.
- **`history` gains two names outside the naming table.**

## References

| What | Where |
|---|---|
| Machines, the task scheduler and campaigns in Go, whose check after a step this changes | RFC-0011 |
| The history seam and the linearizability checker in Go | RFC-0009 |
| Wing and Gong, "Testing and verifying concurrent objects", the scan that the search runs | Journal of Parallel and Distributed Computing 17, 1993, pages 164 to 182 |
| Lowe, "Testing for linearizability", the memo of configurations | Concurrency and Computation: Practice and Experience, 2017, <https://www.cs.ox.ac.uk/people/gavin.lowe/LinearizabiltyTesting/paper.pdf> |
| Herlihy and Wing, "Linearizability: a correctness condition for concurrent objects", the order of two calls in real time | ACM TOPLAS 12(3), 1990 |
