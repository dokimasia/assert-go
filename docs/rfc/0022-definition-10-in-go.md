---
rfc: 0022
title: Definition 10.0.0 in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-22
updated: 2026-10-22
discussion: https://github.com/dokimasia/assert-spec/issues/18
supersedes: RFC-0004 and RFC-0018, in part
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0022: Definition 10.0.0 in Go

## Summary

The module vendors definition 10.0.0. The definition changes six things
for Go:

| Change | Go API |
|---|---|
| An allocation ceiling counts ten times at most, in place of three | `MaxAllocs` and `MaxAllocsWithSetup` of `assert` and `expect`, and `prop.MaxAllocs` |
| A helper ends its call with a fault through the seat | `Fault` and `FaultReporter` of `assert` and `expect`, `Recorder.ReportFault` and `Recorder.Faults` |
| A check that a fault ends ends the call of `Rejects` with that fault | `Rejects` of `assert` and `expect` |
| A test reads the values of a generator for a seed | `prop.Values` |
| A test reads whether its run checks allocation ceilings | `AllocsChecked` of `assert` and `expect` |
| A run of `ForAll` and `Fuzz` takes `Using` | `prop.Using` returns an `Option` |

Every call record states `10.0.0`.

## Motivation

The definition's RFCs argue the first five changes. This RFC fixes their
Go signatures, and the design of the sixth, which changes a decision of
this module.

`Using` is a `FormOption` today, so `ForAll` and `Fuzz` refuse it at
compile time. The definition gives a run of every property the option.
A body of `ForAll` that draws `prop.Of[Order]()` cannot narrow the
`Status` inside each order for that run alone. `RegisterValues` narrows
it for the whole test process, and a second registration of one type
panics. A test binary then cannot import two packages of test helpers that
each register `Status` in `init`.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| Ten counts | `internal/matcher/allocs.go` | Counts a ceiling ten times at most |
| The reason of an unchecked run | `internal/matcher/allocs.go`, `allocs.go`, `expect/allocs.go` | `AllocationsUncounted`, and `AllocsChecked` on both surfaces |
| Faults through the seat | `fault.go`, `expect/fault.go`, `recorder.go` | `Fault`, `FaultReporter`, and the faults that a `Recorder` keeps |
| The fault of a check | `internal/matcher/rejects.go` | Ends the call of `Rejects` with the fault that ends its check |
| The values of a seed | `prop/values.go` | `Values` |
| The generators of a run | `prop/using.go` | `Using` as an `Option`, the generators of one run, and the pick of `Of` |
| Picks and scopes | `internal/prop/engine` | `Picking`, `Within`, `Inversion`, `InvertUsing` and `Settings.Using` |

### Ten counts

`allocCounts` becomes 10. The counts stop at the first count that meets
the ceiling, so a function that meets it costs one count as before. The
runtime builds the cache of a type assertion at a call site that misses
it in about one of 1,024 calls. A function that asserts types for the
first time allocates in those calls, for thousands of calls after the
first.

### The reason of an unchecked run

```go
// AllocsChecked reports whether MaxAllocs, MaxAllocsWithSetup, their
// property forms and the allocation ceilings of a benchmark contract check
// their ceilings in this run. Where they do not, they pass without a count,
// and reason names the build or the run. Where they do, reason is empty.
func AllocsChecked() (checked bool, reason string)
```

`reason` is one of four texts, in this order of precedence:

- `a build with the race detector, msan or asan`
- `a build whose -gcflags turn off optimisation or inlining`
- `a test binary that a mutation run instrumented`
- `a run that writes the test log of go test, which a run with -count=1
  does not write`

`matcher.AllocationsUncounted` returns the reason, and
`matcher.AllocationsCounted` reports whether it is empty. The result is the
same for every call in one run. `expect.AllocsChecked` returns the result
of `assert.AllocsChecked`, because both surfaces declare the same members.

### Faults through the seat

```go
// FaultReporter is a TB that takes a fault as the error it is, rather than
// as the writer's text of it, as a Reporter takes a failure's record.
type FaultReporter = matcher.FaultReporter

// Fault ends the call of a helper with err, a fault: an argument or an
// environment that the helper refuses.
func Fault(tb TB, err error)

// ReportFault keeps err among the faults of the Recorder.
func (r *Recorder) ReportFault(err error, ending bool)

// Faults returns every fault that arrived through ReportFault, in call order.
func (r *Recorder) Faults() []error
```

- `Fault` passes `err` to the first seat of the chain of `tb` that
  satisfies `FaultReporter`, with `ending` true. A chain without one
  receives the writer's text of `err` through `Fatalf`. `Fault` does not
  write a call record, because the call is not a call of an assertion.
- `Fault(tb, nil)` panics, because a nil error is not a fault.
- `expect` declares `Fault` and `FaultReporter` as well, because both
  surfaces declare the same members. A fault ends the call on both
  surfaces, as the fault of an assertion does.
- `ReportFault` of a fault with `ending` true fails the recorder, as
  `Fatalf` does with the writer's text of the fault. A noted fault does not
  fail it.
- `Faults` returns a copy, as `Failures` and `Records` do.

### The fault of a check

The seat of the check that `Rejects` drives keeps the fault that ends the
check, and ends the goroutine of the check, as `Fatalf` does. `Rejects`
then ends its own call with that fault through `Running.Fault`, and returns
nil. Its call record states the verdict `error`. A fault that the check
notes and runs on past passes to the seat of `Rejects`, as `NoteFault`
notes it there.

### The values of a seed

```go
// Values returns n values of g for seed: the values of the random cases of
// seed, in order, without the cases that g rejects.
func Values[T any](g Generator[T], seed uint64, n int) []T
```

- Case k decodes from the random source of seed + k, as a random case of a
  run does, so the values are the ones that the definition's generation and
  shapes vectors state.
- `Values` panics for n below 0, and when g rejects more than ten cases for
  each of the n values. A panic of g passes through.
- It does not run a property. It does not try the simplest case or a
  boundary case. It does not record an assertion or read a variable of the
  engine.

### Using in every run

```go
// Using makes g the generator of T in the run that it is passed to.
func Using[T any](g Generator[T]) Option
```

`Option` implements `FormOption`, so every call of a form with `Using`
compiles as before. Each Using of a run adds a generator, and a later
Using of one type replaces an earlier one.

A run's generators apply to the draws of `Of`:

| Draw | Generator |
|---|---|
| `Of[T]()` of a type that the run states | The stated generator |
| `Of[U]()` of a type that contains a stated type | U read with the stated generators in place of their types |
| `Of[T]()` inside the stated generator of T | T without its own stated generator, with the run's other generators |
| Any other generator, such as `Integer(0, 9)` | The generator itself |

A draw of `Of` inside another generator, such as `List(Of[Status]())`,
takes the run's generators as well. A form's input is such a draw.

The third row keeps a stated generator from drawing itself without end.
`Using(Of[Order]().Filter(valid))` then narrows the orders that the run
derives, and the run's generator of `Status` still applies inside them.

`Of[T]()` reads T without the run's generators, because the call does not
know its run. It panics for a type that the reader refuses, as before. A
draw of `Of[U]()` reads U again with the run's generators, once per run
and type. A read that fails there panics inside the case with the message
of `Of`. Such a read fails for a prop tag on a field whose type the run
states, because the tag then does not apply to any part of the field.

### The engine

`Of` returns a generator of `Picking`. The engine passes `Settings.Using`
to its pick at each decode, and `prop` keeps the generators of one run
there:

```go
// Picking returns g with a decode and an inverse that use the generator
// that pick returns, where pick returns one, and g where it does not.
func Picking[T any](g Generator[T], pick func(using any) (Generator[T], bool)) Generator[T]

// Within returns g with a decode and an inverse that run in place of what
// the run states for the generators of types what using returns.
func Within[T any](g Generator[T], using func() any) Generator[T]
```

- A stated generator of T decodes within the run's generators without T.
  That gives the third row of the table.
- A draw keeps what the run states where the draw runs. The explain phase
  decodes the draw's generator again under it.
- An inverse takes an `Inversion`, which passes what the run states to the
  inverses inside it. The inverse of a picking runs through the picked
  generator. An entry of `Draws`, a stored case of values and an `Example`
  of a form then run backwards through the generator that the run decodes
  with.

```go
// Inversion is what the inverse of a generator passes to the inverse of
// each generator inside it.
type Inversion struct{ using any }

func NewInvertible[T any](id string, decode func(*Case) T,
	invert func(in Inversion, v any) ([]Step, T, error)) Generator[T]
func (g Generator[T]) Inverse(in Inversion, v any) ([]Step, T, error)
func InvertUsing[T any](using any, g Generator[T], value T) ([]choice.Choice, error)
```

`Invert(g, value)` keeps its signature, and runs back in a run that does
not state a generator.

A draw of `Map` over a picking records the integer of the generator that
`Of` read, while the pick can return another generator. The explain phase
does not state a nearest passing value for a draw whose first request has
other bounds than that integer.

### The record

`record.Definition` becomes `10.0.0`. The conformance gate pins
`assert.Fault`, `assert.FaultReporter`, `Recorder.ReportFault`,
`Recorder.Faults`, `assert.AllocsChecked` and `prop.Values` against the
naming table.

### Changesets

```markdown
**Breaking:** Count an allocation ceiling ten times at most, in place of three.
```

```markdown
Add `Fault` and `FaultReporter` to `assert` and `expect`, and `Recorder.ReportFault` and `Recorder.Faults`.
```

```markdown
**Breaking:** End the call of `Rejects` with the fault that ends its check.
```

```markdown
Add `prop.Values` to return the values of a generator for a seed.
```

```markdown
Add `AllocsChecked` to `assert` and `expect` to report whether a run checks allocation ceilings.
```

```markdown
Make `prop.Using` an `Option`, which `ForAll` and `Fuzz` take.
```

```markdown
Implement definition 10.0.0 and write its version into every call record.
```

## Alternatives considered

### A. Keep Using on the forms alone

The Go overlay would declare that a run of `ForAll` and `Fuzz` does not
take `using`.

**Why not:** the definition states the option for every property. A body
of `ForAll` that derives a type would keep the registry as its only way to
narrow a type inside it, and the registry applies to the whole test
process.

### B. Resolve the run's generators where Of is called

`Of[T]()` would return the generator of T under the generators of the run
that calls it.

**Why not:** `Of` does not take the run as an argument, and a body may
build a generator before its run starts. Go does not give a goroutine a
state that `Of` could read to find its run.

### C. Decode a stated generator without any of the run's generators

Inside a stated generator, every draw of `Of` would take the generator
without the run's generators.

**Why not:** `Using(Of[Order]().Filter(valid))` beside a generator of
`Status` would derive orders with every `int` as their status. The
definition applies the generator of a type at every place of the type in
the run's inputs.

### D. Pass the run's generators to every decode

A decode would take the run's generators as a second argument, in place of
the case's.

**Why not:** every decode and every caller of `Decode` would change. The
case is already an argument of every decode.

## Drawbacks

- A decode of a picking locks the case once to read what the run states.
- A run with `Using` reads each type that its draws of `Of` name once
  more, beside the read of `Of`.
- `NewInvertible` and `Inverse` change their signatures in
  `internal/prop/engine`. An inverse in the packages `engine`, `shape` and
  `matching` takes an `Inversion`.
- A draw of `Of` whose read fails under the run's generators fails its
  case with a panic, not before the run.
- A failing allocation check calls the function 1,010 times, in ten counts
  of 101 calls.

## Rules of the standards that this amends

The standard of errors and output gains three panics:

- `Fault` panics for a nil error.
- `Values` panics for a count below 0, and for a generator that rejects
  more than ten cases for each value. A panic of the generator passes
  through.
- A draw of `Of` panics inside its case for a type that the reader refuses
  with the run's generators in place.

## References

| What | Where |
|---|---|
| The issues of definition 10.0.0 | <https://github.com/dokimasia/assert-spec/issues/12>, #14, #15, #16, #17 and #18 |
| The definition's option of a single property | <https://github.com/dokimasia/assert-spec/blob/main/docs/rfc/0011-property-forms-and-derived-inputs.md> |
| The design of Using as a FormOption | RFC-0004 |
| The design of three counts | RFC-0018 |
| jqwik's domain context, the generators of one property | <https://jqwik.net/docs/current/user-guide.html#domain-and-domain-context> |
