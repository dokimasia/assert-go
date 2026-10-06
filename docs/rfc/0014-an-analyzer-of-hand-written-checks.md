---
rfc: 0014
title: An analyzer of hand-written checks
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-10-06
updated: 2026-10-06
discussion: https://github.com/dokimasia/assert-go/issues/19
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0014: An analyzer of hand-written checks

## Summary

A test that states a condition through `True` or `False`, or through a
loop or a flag of its own, hides the assertion that states the
condition. Its failure reports `true` or `false` instead of the values
that differ. A review of two modules' tests replaced 54 calls of `True`
and `False`, 26 sequences of an encode, a parse and an `Equal`, and 17
checks of an ended context with the assertions that state them, each
found by hand.

This proposal adds a module of its own, `go.dokimi.dev/assert/lint`, in
the directory `lint/` of this repository:

- **The package `lint` exports `Analyzer`,** a `go/analysis` analyzer
  named `assertlint`. It reports 13 hand-written checks. For 7 of them it
  suggests a fix that keeps the check's meaning.
- **The command `cmd/assertlint`** runs the analyzer through
  `singlechecker`. `assertlint ./...` reports, and `assertlint -fix ./...`
  applies the fixes. `go vet -vettool` runs it as well.
- **The module requires `golang.org/x/tools`.** The module
  `go.dokimi.dev/assert` keeps its empty requirement list.

## Motivation

### The conditions that `True` and `False` hide

The review counted the conditions of the 54 calls:

| Condition in `True` or `False` | Calls | The assertion that states it |
|---|---|---|
| An order of one value, such as `n > 0` | 21 | `InRange`, `NotEqual` |
| A comparison with nil | 15 | `Nil`, `NotNil` |
| The `ok` of `errors.As`, a seen map or a found flag | 9 | `ErrorAs`, `NoDuplicates` and others |
| `x.IsZero()` | 3 | `Equal`, `NotEqual` |
| `bytes.Equal` or `slices.Equal` | 2 | `Equal`, `NotEqual` |
| A length, such as `len(x) > 7` | 2 | `InRange`, `NotEmpty` |
| `x == a \|\| x == b` | 2 | `Contains` |

`True(t, n > 0, "the count is positive")` fails with the message alone.
`InRange(t, n, 1, math.Inf(1), "the count is positive")` fails with the
count. Most of the conditions are syntactic patterns, and a type checker
tells the safe rewrites from the others.

### A module of its own

`go/analysis`, its test harness and `singlechecker` are packages of
`golang.org/x/tools`. The standards admit a dependency of the module
`go.dokimi.dev/assert` only through an RFC. A module of its own keeps the
requirement out of the module graph of every consumer of the assertions.
A test that imports `assert` loads no package of `x/tools`. A run of the
analyzer loads them.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| The analyzer | `lint/analyzer.go` | `Analyzer`, the walk of a package, and the calls of `assert` and `expect` that it recognises |
| One file per rule | `lint/nil.go`, `lint/compare.go`, `lint/length.go`, `lint/conjunction.go`, `lint/errors.go`, `lint/order.go`, `lint/membership.go`, `lint/loop.go`, `lint/update.go`, `lint/chain.go` | The condition that each rule matches, its diagnostic and its fix |
| The edits | `lint/edit.go` | The text of a call rewritten from the source of its arguments |
| The command | `lint/cmd/assertlint/main.go` | `singlechecker.Main(lint.Analyzer)` |
| The test module | `lint/testdata/` | A module that requires `go.dokimi.dev/assert` through a `replace` of the repository's root, with one package per rule and its golden files |

### The public surface

```go
package lint

// Analyzer reports a check that a test writes by hand and that an
// assertion of go.dokimi.dev/assert states, and suggests the rewrite where
// the rewrite keeps the check's meaning.
var Analyzer = &analysis.Analyzer{
	Name:     "assertlint",
	Doc:      "report hand-written checks that an assertion of go.dokimi.dev/assert states",
	URL:      "https://pkg.go.dev/go.dokimi.dev/assert/lint",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}
```

The analyzer recognises a call by the function that the type checker
resolves, in the package `go.dokimi.dev/assert` or
`go.dokimi.dev/assert/expect`. A fix keeps the qualifier that the call
writes, so a call of `expect` remains a call of `expect`. The analyzer
reads the methods of the chain from the type `expect.Assertion` that the
package under analysis imports, so the rule on chains follows the chain
without a list of its own.

### The rules

| Rule | Matches | Fix |
|---|---|---|
| `nil` | `True` or `False` of `x == nil` or `x != nil`, x a pointer, slice, map, channel, function or `unsafe.Pointer` | `Nil` or `NotNil` |
| `error-nil` | The same over x of an interface type that implements `error` | `NoError` or `HasError` |
| `compare` | `True` or `False` of `a == b` or `a != b`, both of a basic type | `Equal` or `NotEqual` |
| `equal-func` | `True` or `False` of `bytes.Equal(a, b)`, or of `slices.Equal(a, b)` over a basic element type | `Equal` or `NotEqual` with `EquateEmpty()` |
| `length` | `Equal(len(x), n)`, `Length(x, 0)`, `Equal(len(x), 0)`, `NotEqual(len(x), 0)`, and `True` or `False` of `len(x) == n` | `Length`, `Empty` or `NotEmpty` |
| `conjunction` | `True(a && b)` as a statement | One `True` for each operand, with the message of the call |
| `errors-is` | `True` or `False` of `errors.Is(err, target)` | `ErrorIs` or `ErrorIsNot` |
| `errors-as` | `True(errors.As(err, &target))`, and `True(ok)` after `ok := errors.As(err, &target)` | Reported without a fix |
| `order` | `True` or `False` of `<`, `<=`, `>` or `>=` over numbers | Reported without a fix |
| `membership` | `True(x == a \|\| x == b ...)` over one x | Reported without a fix |
| `loop` | A range over a slice whose body is one `NoError` of a call on the element | Reported without a fix |
| `update-flag` | `flag.Bool("update", ...)` in a test file | Reported without a fix |
| `chain` | Two or more consecutive calls of `expect` on one variable, each of a method of the chain | Reported without a fix |

Each diagnostic states the rule and the assertion, such as
`compare: state the comparison with Equal`.

### Why seven rules fix and six report

A fix applies only where the rewrite keeps the check's meaning in every
case:

- **`nil` excludes other interfaces.** `Nil` counts a typed nil in an
  interface as nil, and `==` does not, so `True(x == nil)` and `Nil(x)`
  differ for such an x. An `error` keeps `==`'s meaning through `NoError`,
  which reports a typed nil as an error.
- **`compare` excludes pointers, structs and arrays.** `Equal` compares
  their contents, and `==` compares a pointer by its address.
- **`equal-func` adds `EquateEmpty`,** because `bytes.Equal` and
  `slices.Equal` report a nil slice equal to an empty one.
- **`errors-as` has no fix,** because `ErrorAs[T]` takes the target's type
  as its type argument. A fix must spell that type, and a type that the
  file writes through an alias, or that a short variable declaration
  leaves unwritten, can make the fix fail to compile.
- **`order` has no fix,** because `InRange` states a closed range of
  `float64`. A strict order, and an integer beyond 2^53, have no exact
  rewrite.
- **`membership` has no fix,** because `Contains` takes a slice of x's
  type, whose spelling a fix cannot always write.
- **`loop` has no fix,** because `Total` takes a function of one argument,
  and the loop's call can take more.
- **`update-flag` and `chain` have no fix,** because the rewrite replaces
  statements that the test arranges in its own way.

### The command and its coverage

The command is one statement. Its test builds the command with `-cover`
into a directory of the test, and runs it over a copy of a package of the
test module with `GOCOVERDIR` set to the directory of `-test.gocoverdir`.
`go test -cover` then merges the command's counters with the test's, so
the command's statement counts as covered without a test in the package
`main`. A probe under Go 1.27.1 confirmed the merge: a command built and
run that way reported its one statement covered under `go test
-coverprofile -coverpkg=./...`. The test requires the command's
diagnostics, its exit status 3, and the files that `-fix` writes.

### The repository

- `.ergon.yaml` lists both modules, `modules: [".", "lint"]`, so `ergon
  check` lints, tests and covers `lint` as it does the root.
- The mutation stage mutates only `internal/matcher`, `internal/equality`
  and `conformance`.
- The page of `go.dokimi.dev/assert/lint` on the host `go.dokimi.dev`
  serves the same `go-import` tag as the page of `go.dokimi.dev/assert`.
  The host serves no page for that path today, and a consumer outside
  this repository resolves the module through it.
- A release of the analyzer is a tag `lint/vX.Y.Z`.

### Testing

- One package of the test module per rule, under `lint/testdata/`, whose
  `// want` comments state each diagnostic and whose golden files state
  each fix. `analysistest.RunWithSuggestedFixes` runs each package from
  the spec of its rule's file.
- Every rule has a package of calls that it must not report: an interface
  that is no `error` for `nil`, a pointer for `compare`, a `slices.Equal`
  over structs for `equal-func`, `False(a && b)` for `conjunction`.
- 100% statement coverage of the module, as for the root module.

## Alternatives considered

### A. The analyzer in the module `go.dokimi.dev/assert`

The package `go.dokimi.dev/assert/lint` would be a package of the root
module.

**Why not:** the root module would require `golang.org/x/tools`, and every
consumer's module graph would contain it.

### B. Rules for `ruleguard`

The checks would be rules of `go-ruleguard`, which golangci-lint's
`gocritic` loads from a file.

**Why not:** a rule matches syntax and a few type predicates. The rules
that tell an interface from a pointer, and a basic type from a struct,
need the type checker's results, which an analyzer reads directly.

### C. A plugin of golangci-lint

The module would register the analyzer with
`github.com/golangci/plugin-module-register`, and a custom build of
golangci-lint would run it.

**Why not:** the module would require a second dependency for one runner.
`go vet -vettool` and the command run the analyzer without it, and a
custom build can import `lint.Analyzer` itself.

### D. Fixes for every rule

`order` would rewrite to `InRange` and `membership` to `Contains`.

**Why not:** a rewrite that changes a strict order into a closed range, or
that spells a type that does not compile, turns a lint run into a broken
test. The diagnostic states the assertion, and the author writes it.

## Drawbacks

- **A second module in one repository.** Its tag starts with `lint/`, and
  `ergon` runs every stage once for each module.
- **The module requires `golang.org/x/tools`,** and follows its releases.
- **The host `go.dokimi.dev` needs a page for the module's path.**
- **`errors-as`, `order`, `membership`, `loop`, `update-flag` and `chain`
  report without a fix,** so the author rewrites those calls by hand.
- **The command's test builds the command,** which adds a build of the
  command to the module's test run.

## Unresolved and future work

- Rules for the sequences of an encode, a parse and an `Equal` that
  `RoundTrip` states, and for checks of an ended context that
  `HonoursCancellation` states, are not proposed here. Both span
  statements whose number and shape vary by test.

## References

| What | Where |
|---|---|
| `go/analysis`, `analysistest` and `singlechecker` | `golang.org/x/tools` v0.51.0 |
| testifylint, the checkers `len`, `compares`, `empty`, `error-nil` and `bool-compare` | <https://github.com/Antonboom/testifylint> |
| `go-ruleguard` | <https://github.com/quasilyte/go-ruleguard> |
| Coverage of a program that a test runs, through `GOCOVERDIR` | <https://go.dev/doc/build-cover> |
