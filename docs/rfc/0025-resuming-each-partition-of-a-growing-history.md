---
rfc: 0025
title: Resuming each partition of a growing history
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-11-05
updated: 2026-11-05
discussion: https://github.com/dokimasia/dokimi/issues/58
supersedes: RFC-0013 in part
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0025: Resuming each partition of a growing history

## Summary

`Resume` continues only the search of a check of the whole history, and a check with `Resume` and without `Whole` ends the call with a fault. So a simulation that checks its history after every step searches every call of every partition again at each check, or checks once at the end of a case. This proposal lets `Resume` continue the search of each partition:

```go
// Resume makes the check continue the searches that cp keeps, and keep its
// own searches in cp when it passes.
func Resume[S any](cp *Checkpoint[S]) Option
```

A check with `Resume` and without `Whole` keeps one search for each partition in its checkpoint. The next check of the history continues the search of each kept partition that gained only new calls, and searches every other partition from its first call. It reports what a check of every call reports: its verdict, its steps, its reported partition, its frontier and its final states.

## Motivation

A simulation records the calls of its clients over many keys, and a call can be open for two or more steps of the simulation. A check with `Whole` and `Resume` searches every call in one partition. A crash of a process leaves calls of unknown outcome, and each such call is concurrent with every later call of its partition. In one partition of every key, each such call multiplies the orders that the search tries. A check without `Whole` searches each key apart. Without `Resume`, it searches every call of the history again at each check, so a case of n steps costs about n²/2 steps of the spec. The checks after each step of a machine grew this way before `Resume`.

## Detailed design

### The checkpoint of a partitioned check

A check with `Resume` and without `Whole` keeps in its checkpoint:

- the history, and the number of events that the check read
- one search for each partition, with its calls
- the partition of each key
- the partition of each call that the check read as pending

A check with `Resume` and `Whole` keeps one search of the whole history, as before.

### The next check

The next check reads the events that the history recorded since the kept check:

1. An event that completes a call that the kept check read as pending updates the span of the call, and the partition of the call searches again from its first call. A call that completed as `Fail` leaves its partition, whose other calls then form the partitions of a check of every call.
2. Each new call joins the partitions of its keys, in event order. A call without keys joins every partition. A new call whose keys belong to no partition starts a new partition. A partition that joins another searches again from its first call.
3. The check orders the partitions by their first invocation.
4. A kept partition that neither searches again nor stops fitting its limits grows its search by its new calls, as a check of the whole history grows its search. A kept partition without new calls continues its search without a step.
5. Every other partition searches from its first call.
6. The check sums the steps of the partitions in their order, and reports the first violated partition, else the first undecided one, as a check of every call does.

The new calls of a partition follow every event that its kept search read. So the grown search of a partition takes the steps that a search of every call of the partition takes after the order that the kept search found, as for a check of the whole history.

### The conditions for a search from the first call

| Condition | Scope | Why |
|---|---|---|
| The checkpoint keeps no search | Every partition | The checkpoint is new, or the last check did not pass |
| The checkpoint keeps the searches of another history | Every partition | The calls differ |
| The checkpoint keeps a search of the whole history, or the check states `Whole` and the checkpoint keeps partitions | Every partition | The partitions differ |
| The budget or the memo limit differs | Every partition | A search under other limits stops elsewhere |
| A new event completes a call that the kept search of the partition read as pending | The partition of the call | The call changes: it becomes known, unknown, or leaves the history |
| A new call joins two or more partitions | The joined partition | Its calls interleave the calls of the partitions that it joined |
| The memo contains more configurations than the grown search may store | The partition | A search of every call uses up its memo limit before the order that the kept search found |

After a check that does not pass, and after a check whose spec panics, the checkpoint does not keep a search.

### The public names

```go
// Checkpoint keeps the searches of the last check of a history that passed
// with it, so that the next check of the history continues those searches
// instead of searching from the first call. A check with Whole keeps one
// search of the whole history, and a check without Whole keeps one search
// for each partition. [Resume] passes it to a check. The zero Checkpoint
// does not keep a search.
type Checkpoint[S any] struct{ /* unexported */ }
```

The fault of a check with `Resume` and without `Whole` goes. A check with `Resume` of a history without calls calls the spec's `Initial` under `Whole`, as before, and under `Final`, as every check does. A check with `Resume` and without `Whole` of a history without calls calls nothing else.

### Allocation contract

A partitioned check that continues the searches of the check before it allocates for the events since that check, for the searches that it grows, and for the partitions that it searches again.

## Alternatives considered

### A checkpoint for each key

The caller keeps one checkpoint for each key and checks each partition with `Whole`.

**Why not:** a partition is a set of keys that the calls connect, and a call with two keys joins two partitions. The caller would compute the partitions again, and would aggregate the records of its checks into the record that one check states.

### The partitions of every call at each check

The check computes the partitions of every call of the history at each check, and continues the search of each kept partition whose calls start its new partition.

**Why not:** the check would read every event of the history at each check. Its allocations would grow with the history, where the check of a whole history allocates for the events since the check before it.

## Drawbacks

- **One memo for each partition.** A checkpoint keeps the memo of each partition between checks, the memory of a check of every call.
- **A search from the first call after each completion.** A call that is pending at one check and completes before the next makes its partition search from its first call. In a simulation whose calls are open for two or more steps, each completion of such a call costs a search of its partition from the first call.
- **The bookkeeping of the checkpoint.** The checkpoint keeps the partition of each key and of each pending call.

## Testing

- A property of 300 cases records random histories of up to 30 events over three clients and three keys, with calls without keys, calls that complete as `OK`, `Fail` and `Unknown`, and calls that remain pending, over a register and a register whose write may be lost. After each event it checks the history with `Resume` and without `Whole`, and without `Resume`, and requires the same records and the same final states, on one worker and on two.
- A test for each condition of the table counts the steps of the spec.
- The checks after each of 20 sequential calls over two keys step each call once.

## References

| What | Where |
|---|---|
| The simulation that checks its history after every step | <https://github.com/dokimasia/dokimi/issues/58> |
| Wing and Gong, "Testing and verifying concurrent objects", the scan that the search runs | Journal of Parallel and Distributed Computing 17, 1993, pages 164 to 182 |
| Lowe, "Testing for linearizability", the memo of configurations | <https://www.cs.ox.ac.uk/people/gavin.lowe/LinearizabiltyTesting/paper.pdf> |
