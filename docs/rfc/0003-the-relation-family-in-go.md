---
rfc: 0003
title: The relation family in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-02
updated: 2026-10-02
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0003: The relation family in Go

## Summary

Definition 2.1.0 adds fourteen assertions: the thirteen members of the
relation family and the value assertion `permutation`. This RFC fixes
their Go signatures, the type of each callable, the record that each
failure reports, and the tests that check them against the definition.
Each assertion is one function in `internal/matcher`, wrapped by
`assert` and `expect` as every other assertion is.

## Motivation

The definition states each member's arguments and law in
language-neutral terms. Go decides the rest:

- Whether a callable returns an error beside its value
- Which type parameters a member takes, and their constraints
- How a member tells a panic apart from a test that `Fatalf` ended
- How the corpus calls a member that takes callables

A callable of the wrong type costs a wrapper at every call site. The
callable types in this RFC follow the functions that Go callers already
have.

## Detailed design

### Components

| Component | Responsibility |
|---|---|
| `internal/matcher/relation.go` | The thirteen members, the guard that runs a callable, and the repetition count |
| `internal/matcher/contains.go` | `Permutation`, beside `Contains` |
| `relation.go` and `contains.go` | The aborting wrappers |
| `expect/relation.go` and `expect/contains.go` | The recording wrappers |
| `internal/matchertest/relation.go` and `contains.go` | The shared suites, which drive the matcher and both surfaces |
| `conformance/subject.go` and `conformance/driver.go` | The twelve new subject kinds, and the drivers of the thirteen members |
| `conformance/registry.go` | The invokers of `permutation` |
| `conformance/spec/` | Definition 2.1.0, vendored with `make spec-sync` |

### Signatures

```go
func Idempotent[I, S any](tb TB, call func(I) error, input I, observe func() S, msg string, opts ...Option)
func Accumulates[I any](tb TB, call func(I) error, input I, observe func() int, msg string)
func Deterministic[I, O any](tb TB, call func(I) (O, error), input I, msg string, opts ...Option)
func Commutative[T, R any](tb TB, combine func(a, b T) R, a, b T, msg string, opts ...Option)
func Associative[T any](tb TB, combine func(a, b T) T, a, b, c T, msg string, opts ...Option)
func RoundTrip[I, E any](tb TB, forward func(I) (E, error), inverse func(E) (I, error), input I,
	msg string, opts ...Option)
func StableOrder[T any](tb TB, iterate func() ([]T, error), msg string, opts ...Option)
func NoDuplicates[T any](tb TB, iterate func() ([]T, error), msg string, opts ...Option)
func Monotonic[N cmp.Ordered](tb TB, observe func() N, advance func() error, steps int, msg string)
func Total[I any](tb TB, call func(I) error, domain []I, msg string)
func NotPure[S any](tb TB, observe func() S, fn func(), msg string, opts ...Option)
func FailsAfterClose(tb TB, closer, call func() error, sentinel error, msg string)
func Poisoned(tb TB, induce func(), observe func() error, msg string)
func Permutation[T any](tb TB, got, want []T, msg string, opts ...Option)
```

`expect` declares the same fourteen functions. The completeness gate
checks each arity against the definition. It counts neither the seat nor
the options, and a parameter's type names every type parameter. The
definition's `close` is the parameter `closer` here, because `close` is
a Go builtin.

`Permutation` and the eight members that compare two values as `equal`
does take `opts`. `Accumulates`, `Monotonic`, `Total`, `FailsAfterClose`
and `Poisoned` take no options. The definition gives them no
relaxation.

### The type of a callable

A non-nil error is the failure that the definition names. A panic is a
failure too. Each callable has the type that Go functions of its kind
already have:

| Type | Callables | Reason |
|---|---|---|
| `func(I) error`, `func() error` | `call` of `Idempotent`, `Accumulates`, `Total` and `FailsAfterClose`; `advance`; `closer` | An operation run for its effect returns an error in Go. `f.Close` is a `closer` as it is |
| `func(I) (V, error)`, `func() ([]T, error)` | `call` of `Deterministic`; `forward`; `inverse`; `iterate` | A computation that can fail returns its value and an error. `store.List(ctx)` is an `iterate` inside a one-line closure |
| `func(a, b T) R` | `combine` | A binary operation does not fail in Go. A method expression such as `Counter.Merge` is a `combine` as it is |
| `func() S`, `func() N` | `observe` of `Idempotent`, `Accumulates`, `Monotonic` and `NotPure` | `Pure` reads state through `func() S`, and a test hands the same projection to `Pure`, `NotPure` and `Idempotent` |
| `func() error` | `observe` of `Poisoned` | A reading of a poisoned subject is the error it returns |
| `func()` | `induce`; `fn` of `NotPure` | An induced fault is often itself a failed call, and the member must not count it. `NotPure` takes the arguments of `Pure` |

### Failures of a callable

A guard runs every callable under a deferred `recover`. A non-nil
recovered value is the panic value. `recover` returns nil when
`runtime.Goexit` ends the goroutine, and Go 1.21 and later turn
`panic(nil)` into a `*runtime.PanicNilError`. A callable that calls
`Fatalf` on the test's `*testing.T` ends the test as it does outside the
member. The member then reports nothing.

The failure, or the panic value, goes in the field of the value that the
callable did not return. Every other field of the record is nil:

| Member | Callable that fails | Field that takes the failure |
|---|---|---|
| `Idempotent` | the first `call`, or `observe` after it | `first` |
| `Idempotent` | the second `call`, or `observe` after it | `second` |
| `Accumulates` | `observe` before the first `call`, the first `call`, or `observe` after it | `first` |
| `Accumulates` | the second `call`, or `observe` after it | `second` |
| `Deterministic`, `StableOrder` | the first call or iteration | `first` |
| `Deterministic`, `StableOrder` | any later call or iteration | `second` |
| `Commutative` | `combine(a, b)` | `first` |
| `Commutative` | `combine(b, a)` | `second` |
| `Associative` | either call of `combine` on the left side | `first` |
| `Associative` | either call of `combine` on the right side | `second` |
| `RoundTrip`, `NoDuplicates` | `forward`, `inverse` or `iterate` | `got` |
| `Monotonic` | `observe` or `advance` | `second` |
| `NotPure` | `observe` or `fn` | `got` |
| `FailsAfterClose` | `closer`, or a panic of `call` | `got` |
| `Poisoned` | `induce`, or a panic of `observe` | `got` |

`Total` reports a failing element as its law states: `index` is the
element's position and `got` its error or panic value. An error of
`call` in `FailsAfterClose` and of `observe` in `Poisoned` is the value
that the law examines, and no failure of the member.

### Comparison

The equal-comparing members and `Permutation` compare with the options
that `Equal` uses, widened by the caller's `opts`. `NoDuplicates` and
`Permutation` compare pairs of elements, because a hash needs comparable
elements and `==` differs from `Equal` for pointers, NaN and unexported
fields:

- `NoDuplicates` compares each element with every earlier one, which is
  n(n−1)/2 comparisons for n elements.
- `Permutation` matches each element of `want` with an unmatched equal
  element of `got`, which is at most n·m comparisons. `Equal` is an
  equivalence on the values that equal themselves, so a greedy match is
  exact, and a NaN matches nothing unless `EquateNaNs` applies.

Empty lists compare as `Equal` compares them, so `[]T(nil)` against
`[]T{}` fails unless `EquateEmpty` applies.

### Numbers

`Accumulates` reads an `int`, the type of every count in Go. The changes
are exact. A change beyond the range of `int` is reported as a
`*big.Int` and fails the member, because three readings of `int` cannot
produce two equal changes of that size.

`Monotonic` reads any `cmp.Ordered`. A reading is NaN when it differs
from itself. Strings compare byte by byte, as `<` compares them, which
covers identifiers that sort by time. `steps` below 1 advances nothing,
so the member checks only the first reading.

### Repetitions

`Deterministic` calls its subject 32 times, `StableOrder` iterates 32
times and `Poisoned` takes 32 readings, from one constant in
`internal/matcher`. Each stops at the first result that decides the
verdict.

### Permutation

`Permutation[T any]` requires `want` to have the element type of `got`,
as `Equal` requires `want` to have the type of `got`. The chain declares
no `Permutation`, because the chain's `T` is the whole value and a
method cannot require it to be a slice. `Pairwise` has no chain method
for the same reason.

### Conformance

`conformance.Subject` gains a field for each callable and each input
that a member takes: `Call`, `Input`, `Compute`, `Combine` with `A`, `B`
and `C`, `Forward` and `Inverse`, `Iterate`, `Advance` and `Steps`,
`Domain`, `Closer`, `Use` and `Sentinel`, and `Induce` and `Read`.
`Observe` becomes `func() int`, because the definition states the
subjects `accumulates` and `leaves-state-alone` as integers.
`SubjectDrivers` gains the thirteen members on both surfaces.

The decoded arguments of a `permutation` case are `[]int`, `[]float64`
or `[]any`. Its invokers switch on that type and call
`Permutation[int]`, `Permutation[float64]` or `Permutation[any]`.

`TestSurfaceAborting` and `TestSurfaceRecording` gain a driver for each
new function, and the README gains a section of the reference for the
relations.

### Testing

The suites in `internal/matchertest` drive the matcher, `assert` and
`expect` alike. Each member's suite proves the following:

- The law passes and fails as the definition states, with every field
  of the record.
- The error and the panic of each callable go in the field that the
  failure table names. Every other field is nil.
- A callable that calls `runtime.Goexit` ends the member's goroutine,
  and the member reports nothing.
- A relaxation widens the comparison of each member that accepts one.
- `Deterministic`, `StableOrder` and `Poisoned` fail on a difference in
  the 32nd result and pass on a difference in the 33rd.
- `Accumulates` fails a first call that leaves the count unchanged, and
  fails a change beyond the range of `int`.
- `Monotonic` fails a NaN reading, the first reading included.

`Permutation`'s suite covers multiplicity, an extra element, a missing
one, an int against a float in `[]any`, nil against empty, and NaN.
Every package keeps 100% statement coverage and 100% mutation coverage.

## Alternatives considered

### A. `iter.Seq[T]` for `iterate`

`maps.Keys(m)` would then be an `iterate` as it is.

**Why not:** an iterator fails only by panicking, and a list call in Go
returns `([]T, error)`. A map's keys are one call away:
`slices.Collect(maps.Keys(m))`.

### B. An error from `combine`

Every callable would then report a failure the same way.

**Why not:** a binary operation does not fail in Go, and each one would
need a wrapper. A method expression would no longer fit. A panic still
fails the member.

### C. A type parameter for the integer of `Accumulates`

`Accumulates[I any, N Integral]` would read a counter of any width
without a conversion.

**Why not:** the constraint would be a new exported name on both
surfaces. A change of an unsigned counter can be negative, so `N` cannot
report it. One conversion at the call site costs less.

### D. An untyped `Permutation(got, want any)`

`Contains` takes `any`, and the corpus would call it without a type
switch.

**Why not:** the type parameter refuses a mismatched element type at
compile time, as `Equal` does. The type switch is a few lines in the
conformance package.

### E. A hash for `NoDuplicates` and `Permutation`

A map would make both linear.

**Why not:** a map needs comparable elements and compares with `==`,
which differs from `Equal`.

## Drawbacks

- `NoDuplicates` and `Permutation` are quadratic in the length of the
  list. A list of 1,000 elements costs `NoDuplicates` 499,500
  comparisons.
- `Deterministic` and `StableOrder` call a slow subject 32 times.
- A callable that does not return an error needs a closure that returns
  nil.
- A record that reports a failure of a callable has nil in every other
  field, although the member had read some of those values.

## Open questions

None.

## Unresolved and future work

- The property forms of the members, which RFC-0011 of the definition
  states.

## References

| What | Where |
|---|---|
| The definition, version 2.1.0, and its RFC-0002 | <https://github.com/dokimasia/assert-spec> |
| `runtime.Goexit` | <https://pkg.go.dev/runtime#Goexit> |
| `runtime.PanicNilError` | <https://pkg.go.dev/runtime#PanicNilError> |
| `cmp.Ordered` | <https://pkg.go.dev/cmp#Ordered> |
