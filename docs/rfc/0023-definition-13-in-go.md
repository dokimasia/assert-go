---
rfc: 0023
title: Definition 13.0.0 in Go, and the test of Around on every seat
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-27
updated: 2026-10-27
discussion: https://github.com/dokimasia/assert-go/issues/73
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0023: Definition 13.0.0 in Go, and the test of Around on every seat

## Summary

The module vendors definition 13.0.0. A fault in a case ends the call of
the property, and a property under `Around` runs on every seat that has a
test.

| Change | Go API |
|---|---|
| A fault that ends a call in a case ends the call of the property with that fault | `*prop.Case` implements `assert.FaultReporter` |
| A property under `Around` runs on the seat of `Rejects` and on a `Recorder` | `Recorder.WithTest` and `Recorder.Test` |

Every call record states `13.0.0`.

## Motivation

The definition's RFC-0042 argues the first change. This RFC fixes its Go
design.

The second change is a Go matter. The wrapper of `Around` takes a
`*testing.T`. `ForAll` takes it from the first seat of its seat's chain
that is one. The seat that `Rejects` hands its check does not wrap the
test, and a `Recorder` does not wrap one either. A property with `Around`
ends with a fault on either seat, as assert-go issue 75 reports:

```text
prop.ForAll: Around: the seat is no *testing.T and wraps none
```

`go.dokimi.dev/conform` proves each check of a suite through `Rejects`. It
then applies every other check to the planted defect on a `Recorder`. A
property check whose body waits with `synctest.Wait` needs the bubble of
`Around` in each case, so conform cannot prove it.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| The fault of a case | `internal/prop/engine/case.go`, `prop/case.go` | `ReportFault` on the engine's case and on `prop.Case` |
| The status of a case | `internal/prop/engine/execution.go` | `CaseFaulted` and `Execution.Fault` |
| The fault of a run | `internal/prop/engine/runner.go`, `shrink.go`, `confirm.go`, `campaign.go` | `Result.Fault`, which ends a run in every phase |
| The end of the call | `prop/property.go`, `prop/fuzz.go`, `prop/values.go` | Ends the call of a property with the fault, and panics in `Values` |
| The test of a recorder | `recorder.go` | `WithTest` and `Test` |
| The outer seat of a body | `internal/matcher/seat.go` | `Outer` on the seat of the check of `Rejects` and of an attempt of `Eventually` |
| The test of Around | `prop/property.go` | Finds the test along the chain, through a recorder's test and through the outer seat of a body |

### The fault of a case

```go
// ReportFault ends the case at a fault that ends its call, as a fatal
// record ends it, and the run of the property ends with the fault. It
// passes a noted fault to the seat of the property, and the body runs on.
func (c *Case) ReportFault(err error, ending bool)
```

- The engine's case keeps the first fault that ends a call, and ends the
  goroutine of the body through `runtime.Goexit`. The cleanups run.
- `executionOf` returns `CaseFaulted` with the fault for a case that kept
  one, before every other status. A failure that the body reported before
  the fault does not change the status.
- A noted fault passes to `Settings.Seat` through `matcher.NoteFault`.
  `prop` sets the seat of the property there. A run without a seat drops
  the noted fault.

### The fault of a run

`Result.Fault` is the fault that ended the run, and a faulted run states
nothing else, as a refused run does. Each phase ends at a faulted case:

- The tally of a run ends the run at a faulted case of `Draws`, an
  example, a stored case, the simplest case, a random, prefix or edge case,
  and the case of a token.
- `repeat` stops at a faulted run, as it stops at a failure.
- `confirm` returns the fault of a faulted replay, and `conclude` ends the
  run with it.
- The shrinker keeps the first fault that a charged run returns, and spends
  its budget at once. `settle` stops at that run. `conclude` then ends the
  run with the fault, without an explanation.
- A campaign ends at a faulted case and at a faulted conclusion. The store
  keeps the entries of the failures that it concluded before the fault.

### The end of the call

`property.report` ends the call with `Result.Fault` through
`Running.Fault`, before it reads the outcome. `ForAll`, the forms, the
stored cases of `Fuzz` and each input of `Fuzz` report through it. A
faulted run does not save an entry. The seat receives the fault as the
error it is, so `errors.Is` matches it.

`Values` panics at a case that ends with a fault, with the text of the
fault. A generator of `Composite` receives the case, so it can end a call
with a fault.

### The test of Around

```go
// WithTest makes tb the test that the Recorder runs in, and returns the
// receiver so the call chains onto NewRecorder. Under prop.Around, a
// property on the Recorder passes its wrapper the first *testing.T of the
// chain of tb.
func (r *Recorder) WithTest(tb TB) *Recorder

// Test returns the test that WithTest set, and nil where it set none.
func (r *Recorder) Test() TB
```

`WithTest` takes a `TB`, so the root package does not import `testing`.
A `Recorder` does not wrap the test. The recorder keeps the records, the
faults and the skips of a property on it, and the property writes nothing
to the store or the log of the test.

The seat of the check of `Rejects` and the seat of an attempt of
`Eventually` declare `Outer`, which returns the seat of the assertion.
`ForAll` finds the test of `Around` along the chain of its seat:

1. a seat that is a `*testing.T`
2. the test of a seat with `Test`, searched in the same way
3. the outer seat of a seat with `Outer`, searched in the same way

The search does not unwrap the seat of `Rejects`. `wrap.As` would then find
the name and the cleanups of the outer test, and a property under `Rejects`
would write its counterexamples to the store of that test.

### The record

`record.Definition` becomes `13.0.0`. The naming table gives `WithTest`
and `Test` no row, because the recorder's options are Go names, as
`WithClock` and `WithContext` are.

### Changesets

```markdown
**Breaking:** End the call of a property with the fault that ends a call in one of its cases.
```

```markdown
Add `Recorder.WithTest`, and run a property under `prop.Around` on the seat of `Rejects` and on a recorder of a test.
```

```markdown
Implement definition 13.0.0 and write its version into every call record.
```

## Alternatives considered

### A. Unwrap on the seat of Rejects and on a Recorder

The seat of the check would return the seat of `Rejects` from `Unwrap`, and
a `Recorder` of `WithTest` its test.

**Why not:** `wrap.As` finds every method along the chain. A property under
`Rejects` would write its counterexamples to the store of the outer test,
and a skip on a recorder would skip the outer test.

### B. A Recorder that takes a *testing.T

`WithTest` would take a `*testing.T`.

**Why not:** the root package would import `testing`, and a seat that wraps
a `*testing.T` could not be passed.

### C. A fault that ends the run only before the first failure

**Why not:** the definition ends the run in every phase.

## Drawbacks

- A property whose body ends with a fault ends with the verdict `error` and
  no counterexample. A test that read the counterexample of a fault changes.
- The engine's settings gain a seat, which only noted faults read.
- A bubble under `Rejects` reports a deadlock to the outer test, as a
  bubble of any test reports it.

## Rules of the standards that this amends

The standard of errors and output gains one panic and one fault:

- `Values` panics at a case that ends with a fault.
- A property ends its call with the fault that ends a call in one of its
  cases.

## References

| What | Where |
|---|---|
| The fault of a case | <https://github.com/dokimasia/assert-go/issues/73> |
| The test of Around | <https://github.com/dokimasia/assert-go/issues/75> |
| A case that ends with a fault | <https://github.com/dokimasia/assert-spec/blob/main/docs/rfc/0042-a-case-that-ends-with-a-fault.md> |
| A wrapper around each run of a property | <https://github.com/dokimasia/assert-spec/blob/main/docs/rfc/0031-a-wrapper-around-each-run-of-a-property.md> |
| Seats that wrap a seat | RFC-0019 |
