---
rfc: 0018
title: Definition 8.0.0 in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-11
updated: 2026-10-11
discussion: https://github.com/dokimasia/assert-spec/issues/11
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0018: Definition 8.0.0 in Go

## Summary

The module vendors definition 8.0.0, and its allocation ceilings count as
that version states:

- A ceiling of 0 admits one allocation over the 100 counted calls.
- A count that exceeds its ceiling is taken again, and the assertion
  fails when three counts exceed it.
- The detail of a failure gains `total`, the allocations of the count
  with the lowest total.

The change covers `MaxAllocs` and `MaxAllocsWithSetup` on both surfaces
and their property forms. The public names do not change.

## Motivation

`MaxAllocs` with a ceiling of 0 passes a callable that allocates on 49 of
its 100 counted calls, because it rounds the average to the nearest whole
number. Definition 8.0.0 fails such a callable. It also states the
measurements of this module's own suite that set the slack and the number
of counts.

## Detailed design

### The count

`internal/matcher/allocs.go` counts each call of an assertion up to three
times:

```go
// allocCounts is the most counts that an allocation ceiling takes. The
// assertion fails when none of them meets the ceiling.
const allocCounts = 3

// zeroSlack is the most allocations that the counted calls can make and
// still meet a ceiling of 0. It covers an allocation that the process makes
// while a count runs.
const zeroSlack = 1

// meets reports whether total, the heap allocations of allocRuns counted
// calls, meets ceiling: at most zeroSlack allocations for a ceiling of 0,
// and an average that [perCall] rounds to at most ceiling for any other.
func meets(ceiling, total uint64) bool {
	if ceiling == 0 {
		return total <= zeroSlack
	}
	return perCall(total) <= ceiling
}
```

- `allocs` and `allocsAfter` return the total of one count, which calls
  the callable once to warm it and then counts 100 calls, as today.
- `MaxAllocs` and `MaxAllocsWithSetup` take counts until one meets the
  ceiling, three at most. A count that meets the ceiling ends the call
  with a pass.
- When no count meets the ceiling, the failure states the count with the
  lowest total:

```go
Fail(seat, mode, "max-allocs", msg, map[string]any{
	wantField:  ceiling,
	gotField:   perCall(lowest),
	totalField: lowest,
})
```

- `totalField` is `"total"`, declared beside the other field names. The
  text writer states it after `got`, so a failure of a ceiling of 0 reads
  as `want 0, got 0, total 25`.
- In a build in which `AllocationsCounted` reports false, each assertion
  calls the callable 101 times and passes, as today. It takes one count.

### The doc comments

The doc comments of `assert.MaxAllocs`, `expect.MaxAllocs`, the two
functions with a setup and `matcher.MaxAllocs` state the slack and the
counts. The property forms run `assert.MaxAllocs` and
`assert.MaxAllocsWithSetup`, and their doc comments link to them. The
`# Allocation contract` of each passing call becomes "nothing besides what
the calls of fn allocate", because a passing call can take more than one
count. The README and the testing standard describe the same count.

### The shared cases

`internal/matchertest.RunMaxAllocs` and `RunMaxAllocsWithSetup` gain the
cases of the definition and lose one:

- A callable that allocates on 49 of the 100 counted calls now fails a
  ceiling of 0, with `total` 49. It replaces the case that passed.
- A callable that allocates once in each count passes a ceiling of 0.
- A callable that allocates on 2 of the 100 calls in every count fails a
  ceiling of 0, with `total` 2.
- A callable whose first count allocates 7 times and whose later counts
  allocate once passes a ceiling of 0.
- A callable that allocates 149 times in each count passes a ceiling of
  1, and one that allocates 150 times fails it.
- `OverCeiling` takes the ceiling, the rounded average and the total, and
  states `total` beside `want` and `got`.

A callable that meets its ceiling at the first count is called 101 times
in every build. One that exceeds its ceiling in every count is called 303
times in a build that counts allocations, and 101 times in any other
build.

### The allocation ceilings of the module

The module's own checks with a ceiling of 0 do not change. The
measurements of definition 8.0.0 ran this module's suite four times with
three counts of every check. Six sites whose callables append to a slice
allocate 6 or 7 times in the first count and once in each later count, so
they pass at the second count. Every other site passed at its first
count.

### The record and the store

`record.Definition` becomes `8.0.0`. The test that compares it with the
vendored version covers the change.

### Changesets

```markdown
---
"go.dokimi.dev/assert": minor
---

**Breaking:** Admit one allocation at a ceiling of 0, count up to three times, and state `total` in the failure of `MaxAllocs` and `MaxAllocsWithSetup`.
```

```markdown
---
"go.dokimi.dev/assert": minor
---

Implement definition 8.0.0 and write its version into every call record.
```

## Alternatives considered

### A. Leave the shared cases of definition 6.0.0

The case of 49 allocations would keep passing.

**Why not:** the case contradicts the definition, and the conformance of
this module rests on the shared cases.

### B. Pre-grow the slices of the six sites

The tests of `Observe`, `Note` and `Cleanup` would grow the record before
the count, so the first count would allocate once.

**Why not:** the second count already admits them. A pre-grown record
would also test a state that a caller's first calls never produce.

## Drawbacks

- A failing allocation check takes three counts, 303 calls of the
  callable.
- The detail of every failure of these assertions gains a field, which a
  caller who compares the detail sees.

## References

| What | Where |
|---|---|
| The definition of the count | assert-spec RFC-0030: <https://github.com/dokimasia/assert-spec/blob/main/docs/rfc/0030-an-allocation-ceiling-of-zero.md> |
| The rounding that the count keeps for a ceiling above 0 | assert-spec RFC-0025: <https://github.com/dokimasia/assert-spec/blob/main/docs/rfc/0025-rounding-an-allocation-count.md> |
