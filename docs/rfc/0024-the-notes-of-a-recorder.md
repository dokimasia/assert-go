---
rfc: 0024
title: The notes of a recorder
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-11-02
updated: 2026-11-02
discussion: https://github.com/dokimasia/assert-go/issues/88
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0024: The notes of a recorder

## Summary

`Recorder` gains `Logf` and `Notes`. `Logf` keeps the text of each note,
and `Notes` returns the notes in call order. A note does not fail the
recorder.

| Change | Go API |
|---|---|
| A recorder keeps each note that it receives | `Recorder.Logf` |
| A test reads the notes of a recorder | `Recorder.Notes` |

The definition does not name the notes of a seat, so this RFC decides the
two names.

## Motivation

The module passes a note to a seat through `matcher.Note`, which writes the
text into the log of the first seat of the chain that has `Logf`. Three
calls send notes:

- A failing property sends each note that its failing case logged through
  `Case.Logf`, before its failure.
- An allocation ceiling sends a note when the build counts allocations and
  the test's log alone stops the check.
- `history.Linearizable` sends a note for a spec that states `Equal` and no
  `Hash`.

A `Recorder` has no `Logf`, so it drops all three. A spec that runs a
failing property on a recorder cannot check what the failing case logged:

```go
rec := assert.NewRecorder()
prop.ForAll(rec, "the case fails", func(c *prop.Case) {
	c.Logf("step 1")
	c.Fatalf("stop")
}, prop.Cases(1), prop.Store(""))
```

`rec.Failures()` contains the failure of the property, `rec.Messages()` is
empty, and no method of `rec` returns `step 1`. A failing run of a
simulation of dokimi reports the log of its world after the counterexample
through `Case.Logf`, so the spec of that report needs the notes on a
recorder.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| The notes of a recorder | `recorder.go` | `Logf` keeps a note, and `Notes` returns the notes |

### The API

```go
// Logf keeps a note: the text that format and args state, as fmt.Sprintf
// formats them. A note does not fail the Recorder, and Message and
// Messages do not return it.
//
// Logf makes a Recorder a seat with a log, so a note that the module passes
// to its seat reaches the Recorder: a note of the failing case of a
// property, which the property passes before its failure, and the notes of
// an allocation check and of a linearizability check.
//
// # Allocation contract
//
// A call with a format and no arguments allocates once on average: the
// note. The list of notes grows with each call.
func (r *Recorder) Logf(format string, args ...any)

// Notes returns every note that Logf kept, in call order. The slice is a
// fresh copy.
//
// # Allocation contract
//
// Notes allocates once: the copy.
func (r *Recorder) Notes() []string
```

Each method takes the mutex of the Recorder, as every other method does,
so a body may log on a goroutine of its own.

### What changes for a seat that embeds a Recorder

A seat that embeds a `*Recorder` gains `Logf` as a promoted method. When
such a seat also wraps a test through `Unwrap`, `matcher.Note` stops at the
Recorder, and the test's log no longer receives the notes. The Recorder
keeps them, and `Notes` returns them.

### Testing

`recorder_test.go` checks that `Logf` keeps a note without a failure, that
`Notes` returns the notes in call order, and the allocation ceilings of
both. `prop/property_test.go` checks that a failing property on a Recorder
leaves the notes of its failing case in `Notes`, in the order the case
logged them. The case that checks a seat without a log runs on a seat that
has the three methods of `TB` alone.

## Alternatives considered

### A. The notes inside the failure record

The record of a failing property would carry the notes of its failing case,
as the sentence of a failure in TypeScript and in Rust states them.

**Why not:** the definition fixes the eleven fields of the detail of a
property's record, and every language reports the same record. A field of
notes would differ from the definition. A test also compares records, not
the text of a sentence.

### B. The notes among the messages

`Logf` would append each note to the list that `Messages` returns.

**Why not:** `Messages` returns the messages of the failures that a
recorder received, as the definition's member of the recorder states.
`Failed` and `Message` read the same failures. A note is no failure.

### C. A wrapper of the consumer

A consumer would embed a `*Recorder` in a seat of its own with a `Logf`
that keeps each note.

**Why not:** every consumer that checks a note writes the same wrapper. A
Recorder is the seat that the module offers for reading what a call
reported, and a note is part of what a property reports.

## Drawbacks

- **A Recorder keeps each note**, so its memory grows with each one, as it
  grows with each message.
- **A seat that embeds a Recorder and wraps a test stops passing notes to
  the test's log.** The Recorder keeps them instead.
- **`Recorder` gains two methods**, `Logf` and `Notes`, beside the 21 that
  it declares.

## References

| What | Where |
|---|---|
| The issue | <https://github.com/dokimasia/assert-go/issues/88> |
| `testing.T.Logf`, which keeps the notes of a test | <https://pkg.go.dev/testing#T.Logf> |
| The chain of a seat, which `matcher.Note` walks | [RFC-0019](0019-seats-that-wrap-a-seat.md) |
