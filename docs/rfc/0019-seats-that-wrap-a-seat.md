---
rfc: 0019
title: Seats that wrap a seat
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-10-21
updated: 2026-10-21
discussion: https://github.com/dokimasia/assert-go/issues/58
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0019: Seats that wrap a seat

## Summary

A type that embeds a `TB` has the three methods of `TB` and no others. This
module looks for more methods on a seat: `Report` for the record of a failure,
`ReportFault` for a fault, `Logf` for a note, `Clock`, `Context`, `Name`,
`Cleanup`, and the methods that keep call records. A seat that embeds a `TB`
has none of them. An assertion on such a seat does not write a call record,
and the notes of a property are lost. The wrapped seat receives a failure as
text instead of as a record.

This proposal lets a seat return the seat that it wraps:

```go
// handle is the seat of a check, with the clock that the check controls.
type handle struct {
	assert.TB
	clock *assert.Controlled
}

// Clock returns the clock that the check controls.
func (h *handle) Clock() assert.Clock { return h.clock }

// Unwrap returns the seat of the check.
func (h *handle) Unwrap() assert.TB { return h.TB }
```

The module calls `Helper`, `Fatalf` and `Errorf` of the seat itself. It looks
for every other method on the seat first, then on the seat that `Unwrap`
returns, and so on. The first seat that has the method provides it. A wrapper
declares the methods that it changes, and the seats that it wraps provide the
others.

## Motivation

### A seat that embeds a TB does not write a call record

With v0.3.1, Go 1.27.3 and `DOKIMI_ASSERT_RECORD=1`, one `Equal` writes these
records:

| Seat of the assertion | Records |
|---|---|
| a `*Recorder` | 1 in `Records()` |
| a seat that embeds the `*Recorder` | 0 in `Records()` |
| a `*testing.T` | 1, the line `=== ATTR  TestRecordsOfATest/direct dokimi.assert.1 {…}` |
| a seat that embeds the `*testing.T` as a `TB` | 0 |
| the seat of `Rejects` | 2, the record of `rejects` and the record of `equal` as its run 1 |
| a seat that embeds the seat of `Rejects` | 1, the record of `rejects` |

The `Equal` of the last two rows fails inside `Rejects`.

`record.Of` finds the call records of a seat in two ways. A seat of this module
embeds a `record.Calls`, and gains its unexported method `calls`. A test's seat
declares `Attr`, `Name` and `Output`, and so does a type that embeds a
`*testing.T`. A type that embeds the interface `TB` gains neither. A seat of
another module cannot declare the unexported method, so no method that the
seat declares can pass the records of a `Recorder` on.

### Every other method is lost the same way

| Method | What a seat that embeds a `TB` loses |
|---|---|
| `Report` | A `Recorder` keeps the sentence of a failure and no record |
| `ReportFault` | The wrapped seat receives a fault as text through `Fatalf` |
| `Logf` | The notes of a failing case and the faults of a store are dropped |
| `Clock` | An assertion that waits reads the real clock instead of the seat's |
| `Context` | `assert.Context` returns `context.Background()` |
| `Name`, `Cleanup` | A property has no store, and it claims no entries |
| `StopTimer`, `StartTimer` | `bench.Contract.Excluding` measures the excluded work |

The handle of a check embeds the seat of the check to add methods to it, such
as a clock that the check controls. Such a handle loses every method of the
table.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| The contract of `Unwrap` | `tb.go` | The doc comment of `TB` states it. An `init` function registers the function that calls `Unwrap` |
| The chain of a seat | `internal/wrap/wrap.go` | Keeps the registered function, and finds the first seat of a chain that has a method |
| The lookups | `internal/matcher`, `internal/record`, `prop` and `bench` | Each finds its method along the chain |

### The chain of a seat

The chain of a seat is the seat, then the seat that its `Unwrap` returns, then
the seat that the `Unwrap` of that seat returns, and so on. The chain ends at a
seat without `Unwrap`, and at an `Unwrap` that returns nil.

Each lookup takes the first seat of the chain that has its method:

| Lookup | Method | Without a seat that has it |
|---|---|---|
| The record of a failure | `Report` | The writer's text through `Fatalf` or `Errorf` of the seat itself |
| A fault that ends the call | `ReportFault` | The writer's text through `Fatalf` of the seat itself |
| A fault that does not end the call | `ReportFault` | The writer's text into the log of the first seat with `Logf` |
| A note | `Logf` | Nothing |
| The clock | `Clock` | `assert.System` |
| The context | `Context` | `context.Background()` |
| The call records | A `record.Calls` of a seat of this module, or `Attr`, `Name` and `Output` | No record |
| The store of a property | `Name` | No store |
| The claim of a store | `Cleanup` | No claim |
| The timer of a benchmark | `StopTimer` and `StartTimer` | The timer runs |

The first seat of the chain that takes a record or a fault as a value receives
it. For a chain without such a seat, the seat itself receives the writer's
text. The seat that a lookup finds decides alone: a `Clock` that returns nil
gives `assert.System`, and the lookup does not search the seats after it.

### The public contract

The doc comment of `TB` states the contract:

```go
// TB is the seat that an assertion reports through.
// [testing.T], [testing.B] and [Recorder] satisfy it.
//
// [testing.TB] declares an unexported method, so no type outside the
// standard library implements it. This package declares TB so that any
// type with these three methods is a seat, such as the seat of a
// generated check body.
//
// A seat that wraps another seat declares a method Unwrap() TB, which
// returns the seat that it wraps, or nil. An assertion calls Helper,
// Fatalf and Errorf of the seat itself. It looks for every other method,
// such as Report, Logf, Clock and Context, on the seat and then on each
// seat that Unwrap returns, in order. The first seat that has the method
// provides it, and the call records are those of the first seat that keeps
// call records. A wrapper declares each method that it changes:
//
//	// handle is the seat of a check, with the clock that the check controls.
//	type handle struct {
//	    assert.TB
//	    clock *assert.Controlled
//	}
//
//	func (h *handle) Clock() assert.Clock { return h.clock }
//	func (h *handle) Unwrap() assert.TB   { return h.TB }
//
// The seats that Unwrap returns must not lead back to the wrapper.
type TB interface {
	// Helper marks the calling function as a test helper, so a
	// failure reports its caller's line.
	Helper()
	// Fatalf records a failure and stops the test.
	Fatalf(format string, args ...any)
	// Errorf records a failure and returns.
	Errorf(format string, args ...any)
}
```

### Registration

`assert` imports the internal packages, so no internal package can name
`assert.TB`. A method `Unwrap() assert.TB` satisfies an interface only when the
interface states the same result type. Only `assert` can make the type
assertion to `interface{ Unwrap() TB }`, and an `init` function of `assert`
registers it with `internal/wrap`. `prop` registers its sentences with
`matcher.RegisterSentence` the same way. Every public package imports
`assert`, so the registration runs before any test starts.

```go
// Register makes unwrap the function that returns the seat that a seat
// wraps, and nil for a seat that wraps none. The root package calls it from
// its init function. Call it only while a package initialises.
//
// # Panics
//
// It panics for a nil unwrap and for a second registration, and registers
// nothing then.
func Register(unwrap func(seat any) any)

// Unwrap returns the seat that seat wraps, as the registered function
// returns it, and nil for a seat that wraps none.
//
// # Panics
//
// It panics before a registration. The root package registers its function
// while it initialises, and every program that calls Unwrap imports the root
// package.
func Unwrap(seat any) any

// As returns the first seat of the chain of seat that is a T, and reports
// whether the chain has one. The chain is seat, then the seat that [Unwrap]
// returns for it, then the seat that Unwrap returns for that one, and so on,
// up to the first nil.
func As[T any](seat any) (T, bool)
```

### Allocations

`As` and `Unwrap` do not allocate. The ceilings of the allocation contracts do
not change.

### Testing

- The spec of `tb.go` runs the three rows of a wrapper in the motivation with
  seats that declare `Unwrap`. Each row writes the records of the row of its
  wrapped seat.
- The spec of each lookup adds a wrapper of a seat that has the method. The
  specs of the clock and the context add a wrapper that declares the method
  itself. The specs of `End` and `Fail` add a wrapper that declares `Fatalf`
  alone.
- The spec of `internal/wrap` walks a chain of four seats, stops at an
  `Unwrap` that returns nil, and checks both panics of `Register`.

## Alternatives considered

### A. `TB` as an alias of `matcher.Seat`

`assert` would declare `type TB = matcher.Seat`. An internal package could then
assert `interface{ Unwrap() matcher.Seat }`, and a method `Unwrap() assert.TB`
would satisfy it without a registration.

**Why not:** `go doc` does not list the methods of an alias of an interface of
an internal package. `go doc go.dokimi.dev/assert Reporter` prints the alias
and its doc comment alone. Every seat implements `TB`, and its reader needs the
three methods.

### B. An embedded `TB` that reflection finds

Each lookup would read the fields of the seat's struct and search the value of
an embedded `TB`.

**Why not:** reflection finds a wrapped seat only in an exported embedded
field, and a lookup would allocate on every assertion. A wrapper would have no
way to state that it wraps no seat.

### C. Methods that forward to the wrapped seat

A wrapper would declare `Report`, `Logf`, `Clock`, `Context`, `Attr`, `Name` and
`Output`, and each would call the method of the wrapped seat.

**Why not:** a seat of this module keeps its call records through an unexported
method, which no other module can declare. A wrapper of a `Recorder` or of the
seat of `Rejects` cannot forward its records. Each wrapper would also need a
new method for every method that the module looks for later.

## Drawbacks

- **A wrapper that changes `Fatalf` or `Errorf` alone does not receive the
  failures that a `Reporter` of its chain takes.** Such a wrapper declares
  `Report` as well.
- **A chain that leads back to one of its seats makes a lookup loop forever.**
  `errors.Is` and `http.ResponseController` loop on such a chain as well.
- **The registered function is state of the process.** The `init` function of
  `assert` sets it once. A program that calls an internal package without
  `assert` panics at its first lookup. Every public package imports `assert`,
  so no program of the public API does.
- **A lookup makes one type assertion for each seat of the chain** up to the
  seat that has the method.
- **The module gains the package `internal/wrap`,** with three functions.

## Rules of the standards that this amends

The standard of errors and output names the seat itself in four rules. Each
rule names the first seat of the chain instead:

- The first seat of the chain that implements `Reporter` receives the record
  of a failure.
- The first seat of the chain that implements `matcher.FaultReporter` receives
  a fault as the error it is.
- `matcher.Note` writes a note into the log of the first seat of the chain that
  has `Logf`.
- The first seat of the chain that keeps call records receives a call record.

## References

| What | Where |
|---|---|
| The report of the lost records | <https://github.com/dokimasia/assert-go/issues/58> |
| `http.ResponseController`, which calls the methods of a wrapped `http.ResponseWriter` through `Unwrap() http.ResponseWriter`, since Go 1.20 | <https://pkg.go.dev/net/http#ResponseController> |
| `errors.Unwrap`, `errors.Is` and `errors.As` | <https://pkg.go.dev/errors> |
| Recording every assertion call, and the writer | RFC-0007 |
