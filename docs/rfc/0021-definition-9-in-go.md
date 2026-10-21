---
rfc: 0021
title: Definition 9.0.0 in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-21
updated: 2026-10-21
discussion: https://github.com/dokimasia/assert-go/issues/55
supersedes: RFC-0020
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0021: Definition 9.0.0 in Go

## Summary

The module vendors definition 9.0.0. Its one change for Go is the option
`prop.around`, which the naming table gives Go as `prop.Around`:

```go
func Around(wrap func(t *testing.T, run func(*testing.T))) Option
```

The engine hands each run of a property's body to the wrapper, and the
wrapper runs it inside the environment that the test chooses.
`prop.Around(synctest.Test)` runs each run in a `testing/synctest` bubble of
its own, under `ForAll`, the forms and `Fuzz`. The other changes of 9.0.0
rename TypeScript names and correct Rust names. They do not change a Go
name.

v0.4.0 added `prop.Bubble` for the same need. `Bubble` becomes a deprecated
name that returns `Around(synctest.Test)`. Every call record states `9.0.0`.

## Motivation

`Bubble` runs each run of the body through `synctest.Test`, and it differs
from the definition's option in four rules:

| Rule | `Bubble` | The definition |
|---|---|---|
| The context of a case | Derives from `context.Background()` | Derives from the test that the environment hands `run` |
| Workers | One run at a time | Up to n runs at once under `workers(n)` |
| A wrapper that skips `run` or calls it twice | Cannot happen | Ends the call with a fault |
| A bubble that deadlocks | Fails the case, and the call shrinks it | The environment reports it, and the record does not state it |

A test that states `Around(synctest.Test)` in Go states the option that
`prop.around` states in every other language. The definition rejects an
option for the fake time of one platform, because the fake clocks of the
platforms differ.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| `Around` and `Bubble` | `prop/option.go` | The option, and its deprecated name |
| The wrapper of each run | `prop/property.go` | Hands the wrapper the test, counts the calls of `run`, and keeps the first misuse |
| `ForAll`, the forms and `Fuzz` | `prop/forall.go`, `prop/fuzz.go` | End a call with the fault of a misuse, and hand the wrapper the test of each fuzz input |
| `Settings.Around` | `internal/prop/engine` | Calls the wrapper on a goroutine of its own, and derives the context of each case from its run |

### The option

```go
// Around runs each run of the body inside wrap. wrap calls run once with a
// test, inside an environment that it chooses, and returns once run has
// returned. synctest.Test is such a function, so Around(synctest.Test) runs
// each run of the body in a testing/synctest bubble of its own.
func Around(wrap func(t *testing.T, run func(*testing.T))) Option
```

- The wrapper's type is the type of `synctest.Test`, so a test passes that
  function itself.
- Under `ForAll` and the forms, the wrapper receives the first seat of the
  seat's chain that is a `*testing.T`. A chain without one ends the call
  with a fault before any run: `prop.ForAll: Around: the seat is no
  *testing.T and wraps none`.
- The context of a case derives from the context of the test that the
  wrapper hands `run`. Inside a bubble that context has a `Done` channel that
  belongs to the bubble, so a body that waits on it blocks durably.
- Given more than once, the last wrapper applies. `Around(nil)` panics.

The wrapper runs every run of the body: the case of `Draws`, each stored
case, each generated case, each shrink candidate, each filling that explains
a draw, each case of a campaign, the case of `Replay` and the case of each
fuzz input. The engine's own work runs outside it. That work includes the
decode of an inverse.

### A wrapper that misuses run

`prop` hands the wrapper a function that counts its calls. The first call
runs the body. A second call, and a call after the wrapper has returned,
return without running it. The call then ends with a fault in place of its
record:

```text
prop.ForAll: Around: the wrapper returned without calling run
prop.ForAll: Around: the wrapper called run a second time
```

After the first misuse, `prop` does not call the wrapper again. Each later
case passes without a choice and does not run the body.

### The engine

```go
// Around runs each run of the body, and nil runs it where the runner
// is. It calls run at most once, before it returns. run derives the
// context of the case from ctx and starts the goroutine of the body, and
// it returns once the body and its cleanups have ended. A case whose run
// Around does not call passes without a choice. The runner calls Around
// on a goroutine of its own, so a runtime.Goexit of Around ends that
// goroutine alone, and it raises a panic of Around again.
Around func(run func(ctx context.Context))
```

The runner calls the wrapper on a goroutine of its own because of the ways
in which `synctest.Test` reports a failure:

- `synctest.Test` calls `t.FailNow` on its caller when the test of the
  bubble failed. The race detector fails that test when it reports a race in
  the run. The goroutine of the wrapper ends, and the test is marked failed.
  On a worker of the engine, a `FailNow` on the worker's own goroutine would
  end the worker in the middle of its case.
- `synctest.Test` panics on its caller when every goroutine of a bubble
  blocks, or when a goroutine is still blocked after the body returned. The
  engine raises the panic again on the goroutine of the run. On one worker
  the panic passes through the call, as it passes through a test that calls
  `synctest.Test` itself. On more workers it ends the test process.

### Workers

The engine calls the wrapper for up to n runs at once under `Workers(n)`.
`synctest.Test` is safe to call from more than one goroutine on one test.
Each call makes a test of its own for its bubble. The specs run 50 cases on
four workers under the race detector.

### Fuzz

`Fuzz` hands the wrapper the `*testing.T` of each input. A `*testing.F` has
no `*testing.T` for the wrapper, so the call on f replays no stored case.
The first input that the process runs replays the stored cases before its
own case. The replay is a call of its own on the test of that input, as
under `Bubble`. A misuse ends the call of the input with the fault of
`prop.Fuzz`.

### Bubble

The code standard keeps a removed name for the next minor release. The old
name calls the new one:

```go
// Bubble runs each run of the body in a testing/synctest bubble of its own,
// as Around(synctest.Test) does.
//
// Deprecated: Use Around(synctest.Test), which Bubble returns.
func Bubble() Option
```

A call of `Bubble` gets the behaviour of the definition:

- A bubble that deadlocks panics the call.
- The runs follow `Workers`.
- The context of a case derives from the test of its bubble.

### The record

`record.Definition` becomes `9.0.0`. The conformance gate pins
`prop.Around` against the naming table.

### Changesets

```markdown
---
"go.dokimi.dev/assert": minor
---

Add `prop.Around` to run each run of a property's body inside a wrapper, such as `synctest.Test`.
```

```markdown
---
"go.dokimi.dev/assert": minor
---

**Breaking:** Deprecate `prop.Bubble`, which now returns `prop.Around(synctest.Test)`.
```

```markdown
---
"go.dokimi.dev/assert": minor
---

Implement definition 9.0.0 and write its version into every call record.
```

## Alternatives considered

### A. Keep Bubble as the Go option

**Why not:** it differs from the definition in the four rules of the
motivation. The naming table gives Go `prop.Around`. A test of a property in
Go would state another option than a test of the same property in any other
language.

### B. Remove Bubble at once

**Why not:** v0.4.0 contains it. The code standard keeps a removed name for
the next minor release.

### C. Fail the case of a bubble that deadlocks

The engine would keep the panic of `synctest.Test` as the failure of the
case, and the call would shrink the case. `Bubble` works this way.

**Why not:** the definition leaves the failures of an environment to the
environment. The record would state a failure that no assertion of the body
reported. A shrink of a deadlock also keeps one blocked bubble for each of
its runs, up to the 2,000 runs of the default budget.

### D. Call the wrapper on the goroutine of the run

The engine would call the wrapper where it calls the body today, without a
goroutine of its own.

**Why not:** a `FailNow` of the wrapper on a worker of the engine ends the
worker in the middle of its case. The race detector makes `synctest.Test`
call `FailNow` for a race in a run.

## Drawbacks

- Each run under `Around` costs one goroutine and two channels beside the
  cost of the wrapper.
- A panic of the wrapper on more than one worker ends the test process, not
  the test.
- A `FailNow` of the wrapper does not stop the test. The run goes on, and
  the test fails when it ends.
- A campaign stores each failure as it concludes it, so a misuse after a
  conclusion leaves that entry in the store.
- `Bubble` changes its behaviour in a minor release.

## Rules of the standards that this amends

The standard of errors and output changes two rules about panics:

- The engine raises a panic of the wrapper of `Around` again on the
  goroutine of the run, in place of the panic of `synctest.Test` that a
  property under `Bubble` raised before a run.
- The list of the places of a `recover` names the goroutine of the wrapper
  of `Around`, in place of the bubble of a case under `Bubble`.

## References

| What | Where |
|---|---|
| The definition's option | <https://github.com/dokimasia/assert-spec/blob/main/docs/rfc/0031-a-wrapper-around-each-run-of-a-property.md> |
| The issue that asked for a bubble under `Fuzz` | <https://github.com/dokimasia/assert-go/issues/55> |
| `synctest.Test`, and the test that it makes for a bubble | `src/testing/synctest/synctest.go` and `testingSynctestTest` in `src/testing/testing.go`, Go 1.27.3 |
| The panic of a bubble that deadlocks | `synctestRun` in `src/runtime/synctest.go`, Go 1.27.3 |
| The withdrawn design of `Bubble` | RFC-0020 |
