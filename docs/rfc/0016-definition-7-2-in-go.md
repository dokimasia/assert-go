---
rfc: 0016
title: Definition 7.2.0 in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-07
updated: 2026-10-07
discussion: https://github.com/dokimasia/assert-spec/issues/10
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0016: Definition 7.2.0 in Go

## Summary

Definition 7.2.0 adds the seat member `seat.cancellation`, which Go names
`Context` after the method of `testing.TB`, and the helper
`cancellation-of`, which Go names `Context` as well. This RFC implements
both and decides the Go API that the naming table does not state:

- **`assert.Context` and `expect.Context`** return the context of any
  seat: the one that its method `Context` returns, and
  `context.Background()` for a seat without that method.
- **`Recorder.WithContext`** sets the context of a recorder, and
  **`Recorder.Context`** returns it.
- **The seat of the check of `Rejects`, and of each attempt of
  `Eventually`,** states a context that derives from the context of the
  assertion's seat. The assertion cancels it when the body ends.
- **`prop`** derives the context of a case through the same function as
  `assert.Context`.

## Motivation

The naming table fixes `Context` for the member and for the helper. Three
Go decisions are outside it:

- the builder that gives a recorder its context
- the function of `internal/matcher` that reads the context of a seat,
  which `assert`, `expect`, `prop` and the seats of a body share
- when the seat of a body derives its context, which decides what
  `Rejects` and `Eventually` allocate

## Detailed design

### The API

```go
package assert

// Context returns the context of tb: the one that its method Context
// returns, as *testing.T, *testing.B, *testing.F, *prop.Case and a
// Recorder state one, and context.Background() for a seat without that
// method.
//
// A helper that takes a TB passes the context to the code under test, so
// that code receives the end of the test through any seat.
//
// # Allocation contract
//
// Context allocates nothing besides what the method Context of tb
// allocates.
func Context(tb TB) context.Context

// WithContext makes Context of this Recorder return ctx, and returns the
// receiver so the call chains onto NewRecorder.
//
// # Allocation contract
//
// WithContext allocates nothing.
func (r *Recorder) WithContext(ctx context.Context) *Recorder

// Context returns the context that WithContext set, and
// context.Background() where it set none.
//
// # Allocation contract
//
// Context allocates nothing.
func (r *Recorder) Context() context.Context
```

```go
package expect

// Context returns the context of tb, as assert.Context does.
func Context(tb assert.TB) context.Context
```

`internal/matcher` gains `ContextOf(seat Seat) context.Context`, which
the four packages call, as `ClockOf` reads the clock of a seat. `TB` keeps
its three methods, so every type with `Helper`, `Fatalf` and `Errorf`
remains a seat.

### The seats of a body

The check of `Rejects` and each attempt of `Eventually` run on a seat that
the matcher makes for the body. Each such seat states `Context`:

- Its first call derives the context from `ContextOf` of the assertion's
  seat through `context.WithCancel`, and each call after it returns the
  same context.
- The assertion cancels the context when the body returns, or when a
  failure ends the body's goroutine.
- A first call after the body ended returns a context that is already
  cancelled.

A body that never reads the context derives none, so the allocation
contracts of `Rejects` and `Eventually` keep their counts. A derived
context allocates twice, measured on Go 1.27.1.

### The conformance table

`conformance` pins `seat.cancellation` to `testing.TB.Context`, the method
that the platform's seat declares, as it pins `collector-seat` to
`testing.T`. It pins `cancellation-of` to `assert.Context`. The surface
test lists `Context` among the functions of both surfaces that report
nothing.

### The definition

The module vendors definition 7.2.0 through `make spec-sync`, after
assert-spec publishes it.

### The documents

The package docs of `assert` and `internal/matcher`, and the README beside
`Rejects`, describe the context of a seat.

## Alternatives considered

### A. A named interface for the method

`assert` would declare `type Contexted interface { Context() context.Context }`,
as it declares `Clocked`.

**Why not:** `testing.TB` already declares the method, and a recorder
states its context through a builder. A named type would add a type row to
the naming table, which the other five languages would decline.

### B. A derived context for every body

**Why not:** each attempt of `Eventually` would allocate twice more, also
for a body that reads no context. `Eventually` retries a body until its
timeout, so the cost grows with the attempts.

### C. `assert.Context` alone

**Why not:** the surface test requires the two surfaces to declare the
same members, and a test that imports `expect` alone reads the context
through `expect`.

## Drawbacks

- Two functions and two methods of the recorder join the public surface.
- The seats of `Rejects` and `Eventually` keep a context behind a mutex,
  and a body that reads it after the body ended reads a cancelled context.
- `ContextOf` reads the method `Context` of a `*testing.F` as well. That
  method panics inside a fuzz target, as a direct call of it does.

## References

| What | Where |
|---|---|
| The cancellation handle of a seat | The definition's RFC-0029 |
| Definition 7.0.0 in Go | RFC-0015 |
| `Context` of `testing.TB`, added in Go 1.24 | `api/go1.24.txt` and `src/testing/testing.go` in Go 1.27.1 |
