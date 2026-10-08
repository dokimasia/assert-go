---
rfc: 0006
title: The kind of a shape fault
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-03
updated: 2026-10-03
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0006: The kind of a shape fault

## Summary

`prop.ShapeOf` and `prop.OfShape` return a fault of the kind
`shape.ErrShape` for a shape that the definition's rules refuse. The kind
is a sentinel of an internal package, so a caller cannot name it in
`errors.Is`. This RFC exports the sentinel from `prop` as `ErrShape`.

## Motivation

`errors.md` requires a public package to export the sentinels of the
faults that its functions return. Two exported functions of `prop` return
an error, and their faults are of one kind or of none:

| Function | Fault | Kind |
|---|---|---|
| `ShapeOf` | A type that the reader refuses, at the path of the part | None |
| `ShapeOf` | A part that a registered generator generates | None |
| `ShapeOf` | A constraint of a `prop` tag that the definition refuses | `shape.ErrShape` |
| `OfShape` | A shape file that the definition's rules refuse | `shape.ErrShape` |
| `OfShape` | A map whose key shape decodes to values that no Go map accepts | None |
| Both | A zone that the platform's time-zone database lacks | None |

A caller of `OfShape` reads a shape file at run time, such as one that
another language wrote. The caller can act on a refused file, by
reporting the file or skipping it, and it cannot match the kind today.

The other kinds of the engine leave `prop` through a seat or a panic,
and never as a returned error. A value of `Example` or `Draws` that the
input's generator does not produce ends its run through the seat's
`Fatalf`, with a fault of the kind `engine.ErrCannotInvert`. No exported
function returns that kind, so `prop` exports no name for it.

## Detailed design

```go
// ErrShape reports a shape that the definition's rules refuse: a shape
// file that OfShape reads, or a constraint of a prop tag in a type that
// ShapeOf reads. errors.Is matches every fault of this kind.
var ErrShape = shape.ErrShape
```

`ErrShape` is the internal sentinel itself. `errors.Is(err,
prop.ErrShape)` then matches each fault that the shape package returns,
with no translation at the package's boundary. Its text is the internal
sentinel's: `shape: the shape does not read`. A fault without a reason
states that text.

The `# Errors` sections of `ShapeOf` and `OfShape` name the kind. A unit
test matches a refused shape file with `errors.Is`.

The name states in Go a part of the definition that the naming table
does not name: the kind of a refusal. The code standard requires an RFC
of this repository for such a name, and this RFC decides it. The
completeness gate checks the names of the naming table alone, and it does
not change.

## Alternatives considered

### A. A sentinel of `prop` that wraps the shape's faults

`prop` declares `errors.New("prop: the shape does not read")`, and
`ShapeOf` and `OfShape` translate each fault's kind at the boundary.

**Why not:** two sentinels would state one condition. Every fault that
crosses the boundary would need a copy with the other kind.

### B. Export `engine.ErrCannotInvert` as well

**Why not:** no exported function returns a fault of the kind. A public
name without a caller is a name that no test can show to be right.

### C. A kind for the faults that have none

The reader's refusals and the map key of `OfShape` get kinds of their
own.

**Why not:** a caller fixes its type or its tag for each of them, and
the fault's path names the part. A kind names a condition that a caller
acts on in code, and no caller acts on these in code.

## Drawbacks

- `prop` exports a sentinel whose text starts with `shape:`, the name of
  an internal package.
- A future kind of `ShapeOf` or `OfShape` needs another export, and
  another RFC.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| The rule that a public package exports its sentinels | `docs/standards/errors.md` |
| The rule that an RFC decides a Go name that the naming table does not state | `docs/standards/code.md` |
| `ShapeOf` and `OfShape` | RFC-0004 |
| `shape.ErrShape` | `internal/prop/shape/shape.go` |
