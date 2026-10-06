---
rfc: 0012
title: Definition 6.0.0 in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-06
updated: 2026-10-06
discussion: https://github.com/dokimasia/assert-spec/issues/8, https://github.com/dokimasia/assert-spec/issues/6
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0012: Definition 6.0.0 in Go

## Summary

Definition 6.0.0 adds three accessors to the failure record, `want`,
`got` and `case-failure`. It also rounds the count of `max-allocs`,
`max-allocs-with-setup` and `bench-max-allocs` to the nearest whole
number, with a half rounded up. This RFC implements both in Go and makes
three decisions that the definition leaves to Go:

- **The accessors are methods of the record's one type,** which
  `assert.Failure` and `expect.Failure` alias.
- **`MaxAllocs` counts with a loop of its own,** because
  `testing.AllocsPerRun` returns only the quotient rounded down.
- **The standards' rules on allocation ceilings follow the new count.**

The RFC amends one rule of `docs/standards/code.md` and one of
`docs/standards/testing.md`.

## Motivation

The definition fixes the names of the accessors and the rounding of the
count.

### The choices of Go

- **The type that declares the methods.** The record is a struct of
  `internal/matcher`. `assert.Failure` and `expect.Failure` are aliases of
  it.
- **How `MaxAllocs` keeps the total.** It counts through
  `testing.AllocsPerRun`, which divides as integers. The function returns
  the quotient, and the total that the new rounding needs is gone.
- **What the module's own ceilings measure.** The standard states that a
  ceiling is the count that `testing.AllocsPerRun` measures. The module
  checks its ceilings with `expect.MaxAllocs`.

## Detailed design

### The accessors

```go
// Want returns the field want of the record's detail, and whether the
// record's assertion declares want. A declared want can be nil.
func (f Failure) Want() (any, bool)

// Got returns the field got of the record's detail, and whether the
// record's assertion declares got. A declared got can be nil.
func (f Failure) Got() (any, bool)

// CaseFailure returns the failure record of the failing case, which the
// field failure of a property's record contains, and whether the record
// contains one.
func (f Failure) CaseFailure() (Failure, bool)
```

The methods are declared on `matcher.Failure`. `assert.Failure` and
`expect.Failure` have them without a copy. `go doc` lists no method of an
alias, and the doc comment of each alias lists the three methods with an
example. Each method allocates nothing. `prop`'s writer reads a property's
failing case through `CaseFailure`.

### The count

| Assertion | Reads the counter | Rounds |
|---|---|---|
| `MaxAllocs` | Once before the 100 counted calls and once after them, at a `GOMAXPROCS` of 1 | To the nearest whole number, a half up |
| `MaxAllocsWithSetup` | Before and after each counted call, so the count leaves the setup out | To the nearest whole number, a half up |
| `bench.Contract.MaxAllocs` | Before the first measured iteration and after the last | `math.Round` of the published quotient |
| `bench.Contract.MaxBytes` | The same window | Down, unchanged |

One unexported function, `perCall`, rounds the totals of both test
assertions: `(2a + n) / (2n)` for a total `a` over `n` calls.
`MaxAllocs` keeps a loop of its own. The loop reads the counter as
`testing.AllocsPerRun` does.

`MaxAllocs` no longer calls `testing.AllocsPerRun`, and a call inside a
parallel test no longer panics. Its doc comment states that the test that
calls it does not call `t.Parallel`. The doc comment of
`MaxAllocsWithSetup` states the same.

### Ceilings that change

A ceiling equals the measured count. A ceiling whose average has a
fraction of a half or more rises by one:

| Ceiling | Was | Is | Measured |
|---|---|---|---|
| `NoGoroutineLeaks` through `assert` and `expect` | 174 | 175 | 174 in an ordinary build, 175 under coverage |
| A shrink run whose budget its first candidate spends | 4,308 | 4,309 | 4,308 or 4,309 from run to run |

Every other ceiling of the module passes unchanged in an ordinary build.

### The vendored definition

`conformance/spec` vendors 6.0.0, and every call record states it. The
completeness gate pins the three accessors to `assert.Failure.Want`,
`assert.Failure.Got` and `assert.Failure.CaseFailure`.

### Changes to the standards

`docs/standards/code.md`, Interfaces and dependencies: the core does not
import the package `testing`. The rule replaces this sentence:

> It calls the package `testing` for `AllocsPerRun` alone.

`docs/standards/testing.md`, Benchmarks and allocation ceilings: the
rule on what a ceiling equals becomes:

> A ceiling is the count that its check measures: `testing.AllocsPerRun`
> in the tests of `internal/matcher` and `internal/equality`, and
> `expect.MaxAllocs`, which rounds the average to the nearest whole
> number, everywhere else. It is 0 for a function that allocates nothing.
> For any other function it is the higher of the counts of an ordinary
> build and a coverage build.

## Alternatives considered

### A. Accessors as functions of `assert` and `expect`

`assert.Want(f)`, `expect.Want(f)` and their siblings would read the
fields.

**Why not:** the naming table names members of the failure record. A
function in `assert` and one in `expect` would declare each accessor
twice.

### B. `MaxAllocs` through the count of `MaxAllocsWithSetup`

`MaxAllocs` would call the count of `MaxAllocsWithSetup` with an empty
setup, and the module would keep one loop.

**Why not:** the count would read the counter 200 times instead of twice.
The stack of the goroutine that counts would also gain two frames.
`NoGoroutineLeaks` writes that stack into its goroutine profile, and its
count rose from 174 to 180 through that path, and to 181 under coverage.

### C. A call of `testing.AllocsPerRun` that keeps its panic

`MaxAllocs` would call `testing.AllocsPerRun` on an empty function before
its own count, so that a call inside a parallel test still panics.

**Why not:** the call measures nothing that the assertion uses. A reader
takes it for a second count. `MaxAllocsWithSetup` states the same rule in
its doc comment and has never panicked.

## Drawbacks

- **`MaxAllocs` inside a parallel test counts the other tests'
  allocations** instead of panicking.
- **The module keeps two loops that count.** One reads the counter around
  each call to leave the setup out. The other reads it around all the
  calls.
- **The ceilings of `NoGoroutineLeaks` and of one shrink run rise by
  one.**

## References

| What | Where |
|---|---|
| Accessors of the failure record | The definition's RFC-0024 |
| Rounding an allocation count to the nearest whole number | The definition's RFC-0025 |
| `testing.AllocsPerRun`, its integer division and its panic in a parallel test | Go 1.27.1, `src/testing/allocs.go` |
