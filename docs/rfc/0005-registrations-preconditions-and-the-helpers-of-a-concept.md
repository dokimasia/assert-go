---
rfc: 0005
title: Registrations, preconditions and the helpers of a concept
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-03
updated: 2026-10-03
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0005: Registrations, preconditions and the helpers of a concept

## Summary

Code that RFC-0002 and RFC-0004 accepted breaks three rules of this
module's standards. This RFC amends each rule to state the case that it
excepts, and amends the testing rule of RFC-0004 to match:

- A registration has no benchmark, because a second call for one type
  panics.
- A registration panics on misuse, as a constructor does.
- An internal function panics when its caller breaks a precondition that
  every caller in the module meets.
- A function that belongs to the concept of one file is declared in that
  file, also when other files call it.

The code of the four cases does not change.

## Motivation

A change brings each file that it touches up to the standards. The move
of `prop` to black-box tests touched the files of each case:

| Rule | Standard | Code that breaks it |
|---|---|---|
| Every exported function and method has a benchmark under `bench.Start(b).MaxAllocs(n)` | `testing.md`, and the testing of RFC-0004 | `Register`, `RegisterValues` and `RegisterVariants` |
| Only a constructor panics on purpose | `errors.md` | The panics of the three registrations, and four panics of internal packages |
| `helpers.go` contains the unexported helpers that more than one file of the package calls | `code.md` | The engine's `inverse.go`, whose helpers seven other files of the engine call |

### Registrations

The definition scopes a registration to the test process. A second
registration of one type panics. Without the panic, two packages that
register one type in `init` would depend on the order in which Go
initialises them. A benchmark calls the function under test in a loop,
and the loop's second call of a registration panics. The registry's
tests reached a registry apart from the process's through
`export_test.go`. The testing standard removed export files, so the
tests now register in `init`, as a caller does, and keep the panic of
each misuse for a test to compare.

`errors.md` allows a panic in a constructor alone. A registration panics
for the reason that a constructor panics. A caller registers in `init`
or in `TestMain`, from literals in its source. An error that a
registration returned would have no caller to act on it.

### Preconditions

Four functions of internal packages panic when their caller breaks a
precondition:

| Function | Panics for |
|---|---|
| `alphabet.Rune`, `internal/prop/alphabet/alphabet.go:39` | An index past the end of the default alphabet |
| `random.Source.Below`, `internal/prop/random/source.go:91` | A bound of 0 |
| `random.Source.Coin`, `internal/prop/random/source.go:100` | A numerator above the denominator |
| `coverage.Bound` and `coverage.Decide`, `internal/prop/coverage/check.go:81` | Fewer than one trial, or successes outside [0, n] |

No input from outside the module can trigger these panics:

- A replayed choice comes from a replay token, a store entry or a shrink
  candidate. The engine coerces it into its request's bounds before a
  decoder reads it, through `SequenceBounds.Coerce` and
  `IntegerBounds.Coerce`. An index of the alphabet is then below
  `alphabet.Size`, and the index of a pattern's class is within the class.
- Every caller of `Below` passes a constant, the length of an edge list
  that contains the target, a count that it checked to be positive, or
  the denominator of odds. `engine.Boolean` and `prop.Odds` check the odds
  when they are built.
- A run decides its coverage after `tally.missing`, which ends a run that
  has no valid case. A label counts once in a case, so its count is at
  most the number of valid cases.

Each of these panics marks a bug in the module, and the tests of the
internal package pin it.

### The helpers of the inverse

The engine's `inverse.go` declares the inverse: `ErrCannotInvert`,
`Step`, `Invert`, and the functions that build steps and faults, such as
`stepOf`, `bitStep` and `uninvertible`. The generators of
`collection.go`, `number.go`, `generator.go`, `recursive.go`,
`selection.go`, `text.go` and `case.go` call them. Moving the unexported
ones to `helpers.go` would split one concept over two files, with `Step`
and `Invert` in one and the functions that build a `Step` in the other.

## Detailed design

### `testing.md`

The benchmark rule reads:

> Every exported function and method has a benchmark under
> `bench.Start(b).MaxAllocs(n)`. A package that only test files import,
> such as `internal/matchertest`, has none, because no program runs its
> code. A registration, such as `prop.Register`, has none either: the
> definition allows one registration of a type in a test process, so no
> loop can call it twice.

RFC-0004's rule that every exported function has a benchmark takes the
same exception.

### `errors.md`

The panic rules read:

> - A constructor panics when its arguments state no domain, as
>   `prop.Integer(5, 1)` does. Its message starts with the package, names
>   the generator by the definition's id with its arguments, and states
>   what they lack: `prop: integer(5, 1) states no value`.
> - A registration panics when its arguments register nothing, or when
>   it would change what a read of the registry has returned:
>   `prop.RegisterValues` of no value, a second registration of one type,
>   and a registration after the first run of a property. Its message
>   starts with the package and names the registration with its type:
>   `prop: Register[shop.Status] registers shop.Status a second time`.
> - An internal function panics when its caller breaks a precondition
>   that every caller in the module meets, such as an index of
>   `alphabet.Rune` past the end of the alphabet. No input from outside
>   the module can trigger such a panic.
> - No other code panics on purpose.

### `code.md`

The helper rules read:

> - A function that belongs to the concept of one file is declared in
>   that file, also when other files call it: the steps of an inverse are
>   in the engine's `inverse.go`.
> - `helpers.go` contains the unexported helpers that more than one file
>   of the package calls and that belong to no file's concept. A helper
>   that one file calls is declared in that file.

## Alternatives considered

### A. A registry apart from the process's for the tests

`export_test.go` gives the tests a `NewRegistry` and a `RegisterIn` for
each registration, and each iteration of a benchmark registers in a new
registry.

**Why not:** the testing standard forbids an export file, because it
declares an API that only the tests can call. The benchmark would
measure a registry that no caller has.

### B. Registrations that return an error

`Register`, `RegisterValues` and `RegisterVariants` return an error.

**Why not:** a caller registers in `init`, where the only action on an
error is to panic. Every caller would write that panic.

### C. Faults for the precondition panics

`alphabet.Rune` returns a rune and a bool, and `Below` an error.

**Why not:** every caller checks the argument before the call, so each
new result adds a branch that no input can take. The coverage standard
counts such a branch as dead code.

### D. The helpers of the inverse in `helpers.go`

**Why not:** a reader of the inverse would read two files, and
`helpers.go` would mix the inverse's functions with the package's other
helpers.

## Drawbacks

- Three exported functions have no allocation ceiling. A registration
  runs once per type and process, so a ceiling would protect no path that
  a test repeats.
- A registration's panic in `init` ends the test binary before any test
  runs. `go test` reports the panic and no test of the package.
- A new caller of a function with a precondition panics at run time when
  it skips the check, where a fault would name the input.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| The standards | `docs/standards/code.md`, `testing.md` and `errors.md` |
| The constructors that panic | RFC-0002 |
| The registrations, and the benchmark of every exported function | RFC-0004 |
| The coercion of a replayed choice | `internal/prop/choice/sequencebounds.go` and `integerbounds.go`, `Coerce` |
