---
rfc: 0001
title: The Go assertion library
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-08-30
updated: 2026-08-30
discussion: none
supersedes: none
superseded-by: none
produces-adr: tbd
---

# RFC-0001: The Go assertion library

## Summary

`go.dokimi.dev/assert` implements the standardized assertion set in Go.
The comparison logic lives once, in an internal package, taking a flag
that says whether a failure aborts. Two public packages wrap it: `assert`
aborts, `expect` records and continues. A third package reads the
definition and the corpus and fails the build when this library disagrees
with either.

## Motivation

The standard says every assertion exists in two namespaces under one
name, and that the two namespaces agree about what each assertion means.
Nothing enforces that inside a single library. Two packages holding 38
functions each, written by hand, disagree the first time somebody fixes a
bug in one and not the other.

Go also cannot list a package's functions at runtime. The completeness
gate the standard requires has to get the public surface some other way,
and picking the wrong way makes the gate check a hand-maintained list
instead of the code, which is a list that goes stale.

Both problems belong to Go rather than to the standard. This RFC answers
them.

## Detailed design

### Packages

```
go.dokimi.dev/assert                    aborting surface, TB, Recorder, Rejects
go.dokimi.dev/assert/expect             recording surface, same members
go.dokimi.dev/assert/golden             golden files
go.dokimi.dev/assert/bench              benchmark ceilings
go.dokimi.dev/assert/internal/matcher   comparison logic
go.dokimi.dev/assert/conformance        corpus runner, gate, overlay check
```

`golden` and `bench` are separate because they need the filesystem and
`testing.B` respectively, and because both abort only, so neither has a
recording twin.

`conformance` holds tests rather than a command, so `go test ./...` runs
the gate and the corpus with no extra CI step.

### The seam

```go
// TB is what assertions require of the seat they report through.
// [testing.T], [testing.B] and [Recorder] satisfy it.
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
}
```

`testing.TB` would cover `testing.T` and `testing.B`, but it has an
unexported method, so nothing outside the standard library can implement
it. A generated check body holds exactly one seat and needs to implement
it, so the interface is declared here instead.

`Errorf` is what the recording namespace reports through.

### Comparison logic, once

Each assertion is one function taking a seat and a mode:

```go
// Equal reports a structural difference between got and want.
func Equal[T any](seat Seat, mode Mode, got, want T, msg string, opts ...Option) {
	seat.Helper()
	if diff := cmp.Diff(want, got, Options(opts...)...); diff != "" {
		Report(seat, mode, "%s: (-want +got)\n%s", msg, diff)
	}
}
```

`Report` picks `Fatalf` or `Errorf` from the mode. The two public
packages are wrappers:

```go
// assert
func Equal[T any](tb TB, got, want T, msg string, opts ...Option) {
	tb.Helper()
	matcher.Equal(tb, matcher.Fatal, got, want, msg, opts...)
}
```

### Each surface declares its own chain

A chain is a real struct in each public package, holding the seat and
the value. Its methods name the mode directly:

```go
// assert
type Assertion[T any] struct {
	tb  TB
	got T
}

func That[T any](tb TB, got T) *Assertion[T] {
	tb.Helper()
	return &Assertion[T]{tb: tb, got: got}
}

func (a *Assertion[T]) Equal(want T, msg string, opts ...Option) *Assertion[T] {
	a.tb.Helper()
	matcher.Equal(a.tb, matcher.Fatal, a.got, want, msg, opts...)
	return a
}
```

One shared type carrying a mode field would avoid writing the methods
twice, and a generic type alias would expose it from both packages.
That was the first design, and it fails on documentation: `go doc`
renders `type Assertion[T any] = matcher.Chain[T]` and lists no
methods, because the methods belong to a package under `internal` that
no documentation tool will open. A fluent surface whose methods are
invisible is worse than the duplication it saves, and the cost grows
with every assertion added.

The methods are generated instead, so the duplication is mechanical
rather than maintained by hand.

### The recording package is generated

`expect`'s functions come from `go:generate`. The generator parses
`internal/matcher`, treats every exported function whose first two
parameters are `Seat` and `Mode` as an assertion, and writes one wrapper
each.

Adding an assertion therefore adds its `expect` wrapper with no second
edit. The generator emits the chain methods on the same pass, from the
same discovery.

### Comparison rules

`go-cmp` supplies the comparison. Three options configure it.

Compare unexported fields:

```go
cmp.Exporter(func(reflect.Type) bool { return true })
```

The standard asks for every field the language reaches without unsafe
access, and Go reaches these.

Compare functions by code pointer:

```go
cmp.FilterValues(bothFuncs, cmp.Comparer(sameFunc))
```

`go-cmp` reports two non-nil functions as unequal even when they are the
same function, so this option implements the standard's identity rule
rather than extending it.

Do not equate a nil collection with an empty one. `go-cmp` only does that
when asked, so this library asks for nothing and omits the option.
Callers who want it pass `assert.EquateEmpty()` on the call.

### Types are checked at compile time

`Equal[T any](tb TB, got, want T, ...)` makes a type mismatch a compile
error. The standard requires `1` not to equal `"1"`; in Go that program
does not build.

The corpus states its values as `any`, so its type-mismatch cases run
through `Equal[any]` and exercise the runtime path. The cases that can
only be expressed as a compile failure carry a skip naming Go and the
reason. Go is stricter here than the standard requires, and the skip
records that rather than hiding a gap.

### The completeness gate

Go cannot enumerate package-level functions at runtime, so the gate reads
the surfaces as source:

```go
file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
for _, decl := range file.Decls {
	out = append(out, exported(decl)...)
}
```

It collects every exported package-level declaration and checks each
assertion in the definition against the names it found. Reading the files
means nobody can satisfy the gate with a list that has gone stale.

`golang.org/x/tools/go/packages` would type-check as well as parse, and
would honour build tags. It also lands in the module graph of everything
that imports this library, which is too much to ask of a consumer for a
check only this repository runs. `go/parser` costs nothing and answers
the question. The price is that a surface split across build-tagged
files would be read as one; neither is, and one that started to would
need this reconsidered.

Note that `parser.ParseDir` is deprecated for exactly the build-tag
reason. `ParseFile` is not, so the directory walk is done here.

The corpus needs a second table, because dispatching a case by its
assertion ID also cannot use reflection:

```go
type Invoker func(r *Recorder, args []any, msg string)

var Registry = map[string]Invoker{
	"equal": func(r *Recorder, args []any, msg string) {
		assert.Equal(r, args[0], args[1], msg)
	},
}
```

The gate checks this table covers every assertion, so it cannot fall
behind the public surface it dispatches to.

### The definition is vendored

The definition and corpus are copied into `spec/` and embedded with
`go:embed`. Tests then run with no network and no sibling checkout. A
`make spec-sync` target refreshes the copy.

### Divergences

This library's overlay is empty. Goroutines make `no-task-leaks` real,
`testing.B` makes the allocation ceilings real, and `context.Context`
makes the behavioural assertions real.

An empty overlay is a claim the gate enforces: an assertion this library
stops implementing fails the build until an overlay entry says why.

## Alternatives considered

### A. Hand-write both namespaces

Write `assert` and `expect` as two independent packages.

**Why not:** about 76 functions and 76 methods across two packages,
each pair required to stay identical, with nothing checking that they
do. The first bug fixed in one and not the other is a silent
disagreement between two namespaces the standard says agree.

### B. One package, with the mode as an argument

`assert.Equal(t, assert.Soft, got, want, msg)`.

**Why not:** every call site passes a mode argument, and almost every one
passes the same value. It also loses the chain's main use, because a
reader cannot see from `assert.That(t, x)` whether the chain stops at the
first failure.

### C. One package, with a soft seat wrapper

`assert.Equal(assert.Soft(t), got, want, msg)`, where `Soft` returns a
`TB` that routes `Fatalf` to `Errorf`.

**Why not:** it is the smallest change, and it works. It puts the mode on
the seat rather than on the call, so a helper that receives a seat cannot
tell whether its assertions abort. The standard also asks for two
namespaces by name, and this supplies one.

### D. Depend on the definition as a Go module

Make the definition repository a Go module and import it.

**Why not:** it makes the definition a Go artifact first and a
language-neutral one second, and every other language would then read a
repository laid out for Go's tooling. Vendoring costs a sync target and a
committed copy.

## Drawbacks

**A generated file in the repository.** `expect/expect.gen.go` is
committed, so a reviewer reads generated code in diffs. Regenerating is a
build step someone can forget, though the gate catches the result.

**The gate does not evaluate build tags.** It reads the surfaces with
`go/parser` rather than loading them, so a surface split across tagged
files would be read as one. This buys a module graph holding one
dependency, `github.com/google/go-cmp`.

**Two tables to keep aligned.** The public surface and the corpus
registry both list every assertion. The gate checks the second against
the definition, so they cannot drift silently, but adding an assertion
means editing both.

**The seam has three methods rather than two.** Anything implementing
`TB` by hand gains a method it may not need.

**The vendored definition can go stale.** `make spec-sync` refreshes it,
and nothing forces anyone to run it. A library can be conformant against
a copy of the definition that is three versions old.

## Open questions

- Should `bench.Contract` keep a `Loop()` method shaped as a drop-in for
  `testing.B.Loop`, or take the benchmark function as an argument?
- Does the recording chain report accumulated failures when the chain
  ends, or leave them to `testing.T` to report at test end? `Errorf`
  already defers to test end, so doing nothing is the cheaper answer.

## Unresolved and future work

Replacing the vendored copy with a checked dependency, so a stale
definition fails the build rather than passing quietly, is not proposed
here.

## References

| What | Where |
|---|---|
| go-cmp, comparison options | https://pkg.go.dev/github.com/google/go-cmp/cmp |
| `go/parser`, and ParseDir's deprecation | https://pkg.go.dev/go/parser |
