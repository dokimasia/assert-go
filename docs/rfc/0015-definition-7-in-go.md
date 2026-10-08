---
rfc: 0015
title: Definition 7.0.0 in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-06
updated: 2026-10-06
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0015: Definition 7.0.0 in Go

## Summary

Definition 7.0.0 names the type that a history check takes after the
sequential specification of an object, and adds the method `returned` to
an operation. This RFC implements both in Go and decides the Go names
that the naming table does not state:

- **`history.Spec[S]`** replaces `history.Model[S]`, with the members
  `Initial`, `Next`, `Equal` and `Hash`.
- **`history.Operation`** replaces `history.Op`, with the members `Name`,
  `Args`, `Known` and `Output`, and the method `Returned`.
- **`history.SpecFrom`** replaces `history.ModelFrom`, and
  `stateful.Machine.Spec` replaces `stateful.Machine.Model`.
- **`history.ErrSpec`** replaces `history.ErrModel`, `Span.Operation`
  replaces `Span.Op`, and `conformance.Specs` replaces
  `conformance.Models`.

## Motivation

The naming table fixes the names of the members that every language
shares. Four Go names are outside it: the sentinel of a fault of a spec's
function, the field of a span that contains its operation, `Spec.Hash`,
and the table of named specs that the conformance runner builds. Each
names the old type today.

## Detailed design

### The API

```go
// Spec is the sequential specification of an object: its state before any
// call, and the states that an operation may leave. S is the type of a
// state.
type Spec[S any] struct {
	Initial func() S
	Next    func(state S, op Operation) []S
	Equal   func(a, b S) bool
	Hash    func(state S) uint64
}

// Operation is a call as a spec sees it: its operation and its arguments,
// and its output when the call completed as OK.
type Operation struct {
	Name   string
	Args   []any
	Known  bool
	Output any
}

// Returned reports whether the call may have returned v: true for a call
// that is not Known, and otherwise whether Output equals v as assert.Equal
// compares them, without an option.
func (o Operation) Returned(v any) bool

func SpecFrom(factory func() Subject) Spec[[]Operation]

func Linearizable[S any](tb assert.TB, h *History, s Spec[S], contract string, opts ...Option)

var ErrSpec = errors.New("history: a function of a spec panics or ends its goroutine")
```

`Span` states its call in the field `Operation Operation`.
`stateful.Machine` states its spec in the field `Spec history.Spec[S]`.

### `Returned`

`Returned` compares through the module's one equality, as `SpecFrom` has
compared a subject's output with the recorded one. `SpecFrom` now calls
`Returned`. A call of `Returned` on an output of a basic type allocates
nothing, measured on a known int output.

### The texts

| Text | Before | After |
|---|---|---|
| The fault of a spec without its functions | the model states no Init or no Step | the spec states no Initial or no Next |
| The fault of a function that panics | the model's Step panics on "read" with boom | the spec's Next panics on "read" with boom |
| The fault of Final of another type | Final states *[]string for a model whose states are of type int | Final states *[]string for a spec whose states are of type int |
| The note of a spec with Equal and no Hash | the model states Equal and no Hash, … | the spec states Equal and no Hash, … |
| The panic of a named spec of `conformance` | the register model has no operation "drop" | the register spec has no operation "drop" |

### Files

`history/model.go` becomes `history/spec.go`, and
`conformance/models.go` becomes `conformance/specs.go`, each with its
test file. Each file is named after the concept it declares.

### The documents

The package docs of `history` and `stateful`, the README and the
standards describe a check in the terms of the definition: a
linearization, the real-time order of two calls, and the sequential
specification of an object.

## Alternatives considered

### A. `ErrModel` unchanged

**Why not:** the sentinel would name a type that no longer exists.

### B. `Returned` through `assert.Equal` with the caller's options

`Returned` would take the relaxations of `Equal`, such as `EquateNaNs`.

**Why not:** a spec compares an output with its own state, whose type it
chooses. A spec that needs another equality compares the output itself.

## Drawbacks

- **Every spec that a caller wrote changes:** its type, its two
  functions, and the name of its operation's member.
- **Four fault texts change.** A test that compares a fault's reason
  states the new text.

## References

| What | Where |
|---|---|
| The sequential specification of a history check | The definition's RFC-0026 |
| The history seam and the linearizability checker in Go | RFC-0009 |
| Machines, the task scheduler and campaigns in Go | RFC-0011 |
