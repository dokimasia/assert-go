---
rfc: 0020
title: Properties in a synctest bubble
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-10-21
updated: 2026-10-21
discussion: https://github.com/dokimasia/assert-go/issues/55
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0020: Properties in a synctest bubble

## Summary

A body that calls `synctest.Wait` has no fuzz target. `Fuzz` runs the body of
each input outside any `testing/synctest` bubble, where `synctest.Wait`
panics. `ForAll` runs such a body when the test calls it inside
`synctest.Test`, and every case of the call then shares one bubble.

This proposal adds an option to `prop`:

```go
// Bubble makes each run of the body run in a testing/synctest bubble of its
// own.
func Bubble() Option
```

Under `Bubble`, each run of the body runs through `synctest.Test` with the
`*testing.T` of the test. `ForAll` and the forms take it from their seat or a
seat that it wraps, and `Fuzz` takes the test of each input. Each run starts at
the bubble's first instant, midnight UTC 2000-01-01, without a goroutine of an
earlier run. A run whose goroutines all block fails its case, and the call
shrinks that case as it shrinks any failure. One property then serves `ForAll`
and `Fuzz` under the same options.

## Motivation

### A body that waits for its goroutines has no fuzz target

With assert v0.3.0 and Go 1.27.2, in a module of its own:

```go
func waitSteps(c *prop.Case) {
	n := c.Draw(prop.Integer(0, 3), "n")
	done := make(chan int)
	go func() { done <- n }()
	synctest.Wait()
	assert.Equal(c, <-done, n, "the goroutine sends the drawn number")
}

func TestWaitInBubble(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		prop.ForAll(t, "a body waits for its goroutine", waitSteps)
	})
}

func FuzzWait(f *testing.F) {
	f.Add([]byte{1})
	prop.Fuzz(f, "a body waits for its goroutine", waitSteps)
}
```

`go test -run 'TestWaitInBubble|FuzzWait' -v .` passes the test and fails the
seed of the fuzz target:

```text
--- PASS: TestWaitInBubble (0.00s)
=== RUN   FuzzWait/seed#0
    a body waits for its goroutine: counterexample after 0 valid and 0 rejected cases, seed 440037688019642277
      n: 0, any value fails
    failure at synctest.go:284: goroutine is not in a bubble: panic string, stack goroutine 45 [running]:
--- FAIL: FuzzWait (0.00s)
```

The machine of a rate limiter whose waiting calls block on the timers of a
fake clock is such a body. Every step of the machine ends with
`synctest.Wait`, so each call that the limiter grants has returned before the
invariant runs. The machine runs under `ForAll` inside `synctest.Test`, and no
fuzz target can run it.

### One bubble for a whole call

A test that calls `ForAll` inside `synctest.Test` runs every case, every replay
and every shrink candidate in one bubble, on one fake clock. Each case starts
at the instant at which the case before it ended, so the replay that confirms
a failure starts at another instant than the case that failed. A goroutine
that a case leaks remains blocked until the call ends. `synctest.Test` then
reports the leak, and no record states which case leaked it.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| `Bubble` | `prop/option.go` | The option |
| The runs of the body | `internal/prop/engine/execution.go` | Starts the goroutine of each run inside `Settings.Around`, and fails the case of a bubble that deadlocks |
| The runs of a property | `prop/property.go` | Finds the `*testing.T` along the chain of a seat, and sets an `Around` that runs `synctest.Test` with it |
| `ForAll` and the forms | `prop/forall.go` | End the call with a fault for a seat without a `*testing.T` |
| `Fuzz` | `prop/fuzz.go` | Runs the runs of each input on the input's test, and replays the stored cases on the first input |

### The runs in a bubble

Each of these runs of the body runs in a bubble of its own:

- the case of `Draws`, each example and each stored case
- the simplest case, and each random, prefix and edge case
- each run that `Case.Repeat` asks for
- the replay that confirms a failure, each shrink candidate and each filling of
  the explanation
- each case of a campaign

A decode that the engine runs without the body, such as the inverse of a
generator, runs outside any bubble.

The budget of a campaign and the time of a shrink are measured in real time.
The clock of a bubble does not move them.

### The test of the bubble

`ForAll` and the forms pass `synctest.Test` the first seat of their seat's
chain that is a `*testing.T`, the one type that `synctest.Test` takes. A seat
without one, such as a `Recorder`, a `*testing.B` or the seat of `Rejects`,
ends the call with a fault before any run:

```text
prop.ForAll: Bubble: the seat is no *testing.T and wraps none
```

`Fuzz` passes the `*testing.T` of each input.

### The context of a case

Under `Bubble`, the context of a case derives from `context.Background()`
inside the case's bubble. A context of the test is outside the bubble. When no
code has read the `Done` channel of the test's context yet, a case that derives
its context from it creates that channel inside the bubble. The end of the test
then closes the channel from outside the bubble, and the runtime stops the
process with "close of synctest channel from outside bubble".

### One run at a time

The runs of a call under `Bubble` run one at a time on the goroutine of the
call, whatever `Workers` states. `synctest.Test` calls `t.FailNow` when the
test of the bubble failed, as it does when the race detector reports a race in
the run. `FailNow` ends the goroutine that calls it, so it must run on the
goroutine of the test. A run on more workers reports what a run on one reports,
so `Workers` changes the time of a run alone.

### A bubble that deadlocks

`synctest.Test` panics in two cases:

- every goroutine of the bubble blocks, with "deadlock: all goroutines in bubble
  are blocked"
- a goroutine is still blocked when the body has returned, with "deadlock: main
  bubble goroutine has exited but blocked goroutines remain"

The engine fails the case of such a run with the panic, as it fails the case of
a body that panics. The call then replays, shrinks and explains the case. The
identity of the panic is its type and the line of the runtime that raises it,
so the deadlocks of one property are one failure. The goroutines of a bubble
that deadlocked remain blocked until the process ends.

The engine raises again on the goroutine of the call a panic that
`synctest.Test` raises before the run starts, such as the panic of a call
inside a bubble.

### The stored cases of Fuzz

Under `Bubble`, `Fuzz` replays the stored cases inside the target, on the first
input that the process runs and before the case of that input, because a
`*testing.F` has no `*testing.T` before its target runs. The replay is a call
of its own on the test of that input. It writes the record and notes the
faults that a replay on f writes and notes. A failing stored case fails that
input with its record, as found, and the case of the input does not run. The
call on f replays no stored case, and notes the faults of the files of the
store that the run skips.

Under `go test -fuzz`, each worker process replays the stored cases on its first
input. The coordinator sends the seed corpus before it generates an input, and
it reports a seed that fails as a failure of the seed corpus. Whenever the
corpus has a seed, a failing stored case fails a seed, and the coordinator does
not write a new failing input.

### The setting of the engine

```go
// Around runs each run of the body, and nil runs it where the runner
// is. It calls run once, and returns once run has returned. The
// goroutine of the body starts inside run and ends before run returns,
// its cleanups included. A panic of Around after it called run fails the
// case as a panic of its body does. The runner raises a panic of Around
// before run again. prop runs each run in a testing/synctest bubble of
// its own through it.
Around func(run func())
```

`prop` passes the function that runs `run` in a bubble on t:

```go
s.Around = func(run func()) {
	synctest.Test(t, func(*testing.T) { run() })
}
```

### The public name

```go
// Bubble makes each run of the body run in a testing/synctest bubble of its
// own, through synctest.Test with the *testing.T of the test. ForAll and
// the forms take the first seat of the seat's chain that is a *testing.T,
// as [go.dokimi.dev/assert.TB] states. [Fuzz] takes the test of each input.
// Each run starts at the bubble's first instant, midnight UTC 2000-01-01,
// without the goroutines of an earlier run. The case's context derives from
// context.Background() inside the bubble, because a context of the test is
// outside it.
//
// The runs of a call under Bubble run one at a time on the goroutine of the
// call, whatever [Workers] states, because synctest.Test fails the test
// through FailNow. A run whose goroutines all block, or that leaves a
// goroutine blocked when its body returns, fails its case with the panic of
// synctest.Test, and the call shrinks that case as it shrinks any failure.
// A seat that is no *testing.T and wraps none ends the call with a fault. A
// call under Bubble must not run inside a bubble, where synctest.Test
// panics.
//
// A *testing.F has no *testing.T before its target runs, so Fuzz under
// Bubble replays the stored cases on the first input that the process runs,
// before the case of that input.
func Bubble() Option
```

`Bubble` is a Go name that the naming table does not state. The definition
fixes what a run reports, and a bubble does not change it.

### Testing

- A body that waits for its goroutine with `synctest.Wait` passes under
  `ForAll` and under each input of `Fuzz`.
- Each run reads midnight UTC 2000-01-01 at its start.
- A body whose goroutines all block fails with the panic of the deadlock, and
  its counterexample is the smallest case.
- A body that leaves a goroutine blocked fails with the panic of the leak.
- A `Recorder` as the seat ends the call with the fault, and a wrapper of a
  `*testing.T` runs its cases in bubbles.
- Under `Workers(4)`, each run starts after the run before it has ended.
- `Fuzz` replays a stored case on its first input, and a failing stored case
  fails that input.

## Alternatives considered

### A. One bubble for each call of ForAll and each input of Fuzz

**Why not:** the clock and the goroutines of one run remain for the next. A
replay and each shrink candidate would start at another instant than the case
that failed. A goroutine that a case leaks would be reported when the call
ends, without its case.

### B. Fuzz without its stored cases under Bubble

`Fuzz` would not replay the stored cases under `Bubble`. The fuzz corpus under
`testdata/fuzz` would keep each failure that `go test -fuzz` finds.

**Why not:** a fix can leave the smallest case of a failure failing while the
input as found passes. The fuzz corpus keeps the input as found. The store
keeps the smallest case. Its replay reports such a fix.

### C. A fault for Bubble with Workers above 1

**Why not:** a run on more workers reports what a run on one reports. A fault
would make a property that serves `ForAll` and `Fuzz` under the same options
drop `Workers` for both.

### D. A test that the case hands its body

`Case` would return the `*testing.T` of the test. The body would call
`synctest.Test` itself.

**Why not:** a draw, a rejection and a fatal failure end the goroutine that
makes them through `runtime.Goexit`. The goroutine that runs the function of
`synctest.Test` must not end that way. `testing` reports a test whose goroutine
calls `runtime.Goexit` outside `FailNow` as a test that panicked.

## Drawbacks

- **A deadlock keeps the goroutines of its bubble.** A shrink of a deadlock
  keeps a bubble for each of its runs, up to the 2,000 runs of the default
  shrink budget.
- **A bubble costs each run about 1.9 µs, 16 allocations and 1.9 KB.** A
  passing call of 136 runs of a body that draws one integer took 471 µs and
  2,685 allocations, against 212 µs and 507 allocations without `Bubble`,
  under Go 1.27.3 at a `GOMAXPROCS` of 4.
- **`Workers` has no effect under `Bubble`.**
- **The context of a case has no value of the test's context.**
- **`Fuzz` reports a failing stored case on its first input.** It registers the
  target before that replay, and the call on f counts no case.
- **A call under `Bubble` cannot run inside a bubble,** because `synctest.Test`
  panics there.

## Rules of the standards that this amends

The standard of errors and output gains two rules about panics:

- A property under `Bubble` raises again a panic of `synctest.Test` that
  refuses a run before the run starts, on the goroutine of the call.
- The list of the places of a `recover` gains the bubble of a case under
  `Bubble`, whose deadlock fails the case.

## References

| What | Where |
|---|---|
| The report of the panic under `Fuzz` | <https://github.com/dokimasia/assert-go/issues/55> |
| `testing/synctest` | <https://pkg.go.dev/testing/synctest> |
| The budget of a campaign in real time inside a bubble | <https://github.com/dokimasia/assert-go/issues/47> |
| Seats that wrap a seat, whose chain `ForAll` searches for the `*testing.T` | RFC-0019 |
| The property engine in Go | RFC-0002 |
