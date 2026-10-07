---
rfc: 0001
title: The Go assertion library
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-08-30
updated: 2026-10-06
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
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
Nothing enforces that inside a single library. Two packages that each
contain the comparison logic of every assertion disagree the first time
somebody fixes a bug in one and not the other.

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
`testing.B` respectively. Neither has a recording twin: a golden
comparison stops the test, and a benchmark contract reports every
exceeded ceiling through the recording surface when it ends.

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

Each package writes its own methods instead. A method is a few lines
that name the mode and call the core, so the duplicated code contains no
comparison logic.

The chain offers the 15 assertions that examine a value of any type, such
as `Equal` and `Contains`, and the four error assertions `ErrorIs`,
`ErrorIsNot`, `NoError` and `HasError`. `That` is generic over the value,
and Go cannot restrict a method to a `T` of `error`, so an error method
over a value that is no error fails, as `Length` fails over a value that
has no length.

### The recording package mirrors the aborting one

`expect` has one function for each function in `assert`, written beside
it by hand. Each calls the same matcher function, in the recording mode
where its counterpart uses the aborting mode. The chain methods follow
the same rule.

Three checks keep the packages in step:

- The conformance gate fails the build when either surface has a member
  the other lacks. `expect` declares the seat, the record and the clocks
  as aliases of the types of `assert`, and the constructors and `Rejects`
  as functions of its own, so a test that imports `expect` alone states
  everything that a test of `assert` states.
- The shared suites in `internal/matchertest` drive every case through
  the core, `assert` and `expect` alike. A wrapper that calls the wrong
  comparison fails a shared case.
- A conformance test calls each `expect` function and chain method
  with an input it rejects. The test fails when a member reports
  through `Fatalf`, reports nothing, or reports as another member. It
  lists the functions from the source and the chain's methods by
  reflection, so a new member that the test does not call fails it.

Adding an assertion therefore takes a wrapper in each package, and the
gate fails until both exist.

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
error. The standard requires `1` not to equal `"1"`. In Go, a program
that compares the two does not build.

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

The gate also reads the parameters of each exported function and method.
It checks the arity that the definition states on both surfaces. The
arity counts the parameters after the seat, without a variadic one. It
also counts each type parameter that no parameter's type names, because
a caller states that one: `ErrorAs[*fs.PathError](t, err, msg)` has an
arity of three. The gate looks a method up by its qualified name, such
as `Contract.MaxAllocs`, and fails on a renamed method.
`NoGoroutineLeaks` is excused with its reason. It returns the check that
ends the scope, and the definition counts the scope as an argument.

`golang.org/x/tools/go/packages` would type-check as well as parse, and
would honour build tags. It would also enter the module graph of every
module that imports this library, for a check that only this repository
runs. `go/parser` is part of the standard library. It does not evaluate
build tags, so it would read a surface split across build-tagged files
as one surface. Neither surface is split that way, and splitting one
would reopen this choice.

Note that `parser.ParseDir` is deprecated for exactly the build-tag
reason. `ParseFile` is not, so the directory walk is done here.

The corpus needs a second table, because dispatching a case by its
assertion ID also cannot use reflection. The table has one invoker for
each form a caller writes: the function of each surface, and the chain
method of each surface where the chain declares one.

```go
type Invoker func(tb assert.TB, args []any, msg string, opts []assert.Option)

var Registry = map[ID]map[Form]Invoker{
	"equal": {
		AbortingCall:   func(tb assert.TB, a []any, m string, o []assert.Option) { assert.Equal(tb, a[0], a[1], m, o...) },
		RecordingCall:  func(tb assert.TB, a []any, m string, o []assert.Option) { expect.Equal(tb, a[0], a[1], m, o...) },
		AbortingChain:  func(tb assert.TB, a []any, m string, o []assert.Option) { assert.That(tb, a[0]).Equal(a[1], m, o...) },
		RecordingChain: func(tb assert.TB, a []any, m string, o []assert.Option) { expect.That(tb, a[0]).Equal(a[1], m, o...) },
	},
}
```

The corpus test runs every case through every form in the table. It
requires both function forms for an assertion that a case states values
for. It also requires each chain form whose chain declares the method. A
form that the surfaces declare and the table omits fails the test. A
case's options are relaxation ids, and a second table maps each id to
its `Option`. The record of a failing case states the case's assertion
and the message unchanged. It contains exactly the fields that the
assertion declares.

### The definition is vendored

The definition and corpus are copied into `spec/` and embedded with
`go:embed`. Tests then run with no network and no sibling checkout. A
`make spec-sync` target refreshes the copy.

### Divergences

This library's overlay declares no divergence. Goroutines make
`no-task-leaks` real, `testing.B` and `testing.AllocsPerRun` make the
allocation ceilings real, and `context.Context` makes the behavioural
assertions real.

An overlay without divergences is a claim the gate enforces: an
assertion this library stops implementing fails the build until an
overlay entry says why.

The overlay does state limits. In a build with the race detector, msan
or asan, in one whose `-gcflags` turn off optimisation or inlining, and
in a test binary that a mutation run instrumented, no allocation ceiling
is checked, because those builds allocate differently from the one that
ships. A run that writes the test log of `go test`, which `go test`
passes to every run whose result it can cache, checks no ceiling either,
because the log allocates in some calls of package `os`.

## Alternatives considered

### A. Two independent namespaces

Write `assert` and `expect` as two independent packages, each with its
own comparison logic.

**Why not:** every comparison exists twice, and each pair must stay
identical with nothing checking that it does. The first bug fixed in one
and not the other is a silent disagreement between two namespaces the
standard says agree.

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

### E. Generate the wrappers

Generate `expect`'s functions and both chains from `internal/matcher`
with `go:generate`, one wrapper for each exported function whose first
two parameters are `Seat` and `Mode`.

**Why not:** each wrapper is a few lines with no logic, and the gate
already fails on a missing one. A generator adds a committed generated
file that reviewers read in diffs, and a build step someone can forget.
A generator would make each wrapper's mode correct by construction. A
conformance test checks the mode of every wrapper of both surfaces
instead.

## Drawbacks

**Every wrapper is written twice.** Each assertion has a wrapper in
`assert` and one in `expect`, and each chain method has one in each
chain. The gate checks that both exist, and the shared suites check what
each reports. Two conformance tests drive every reporting member of each
surface with an input it rejects. They require `Fatalf` from `assert`
and `Errorf` from `expect`. The drivers are a third copy of the member
list. A member added without a driver fails those tests.

**The gate does not evaluate build tags.** It reads the surfaces with
`go/parser` rather than loading them, so a surface split across tagged
files would be read as one. This buys a module graph holding one
dependency, `github.com/google/go-cmp`.

**Two tables to keep aligned.** The public surface and the corpus
registry both list every assertion, and the registry lists each form of
it. The corpus test checks the registry against the definition and the
surfaces, so they cannot drift silently, but adding an assertion means
editing both.

**The seam has three methods rather than two.** Anything implementing
`TB` by hand gains a method it may not need.

**The vendored definition can go stale.** `make spec-sync` refreshes it,
and nothing forces anyone to run it. A library can be conformant against
a copy of the definition that is three versions old.

## Open questions

None. The two this RFC carried were settled by building it.

`bench.Contract` keeps `Loop()` as a drop-in for `testing.B.Loop`, so a
benchmark reads the same with a contract as without one:

```go
for c.Loop() {
    _, _ = store.Get(ctx, id)
}
```

Taking the body as an argument would have measured the same thing and
read as a different kind of code. Keeping the loop shape is why `B`
declares `Loop` at all.

The recording chain reports nothing of its own at chain end. Every
method reports through `Errorf` as it runs, and `testing.T` collects
those and prints them when the test ends. A summary at chain end would
be a second copy of what the framework already prints.

## Unresolved and future work

Replacing the vendored copy with a checked dependency, so a stale
definition fails the build rather than passing quietly, is not proposed
here.

## References

| What | Where |
|---|---|
| go-cmp, comparison options | https://pkg.go.dev/github.com/google/go-cmp/cmp |
| `go/parser`, and ParseDir's deprecation | https://pkg.go.dev/go/parser |
