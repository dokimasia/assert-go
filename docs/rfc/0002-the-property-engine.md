---
rfc: 0002
title: The property engine in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-01
updated: 2026-10-01
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0002: The property engine in Go

## Summary

`go.dokimi.dev/assert/prop` implements `prop-for-all`, the property check
that version 1.2.0 of the definition adds. A body draws its inputs from
generators, the engine runs it over generated cases, and a failing case
shrinks by the definition's passes. The definition fixes every algorithm
and pins each with a vector. This library translates the definition's
executable reference into Go, and the conformance package runs every
vector on every test run. The module gains no dependency.

This proposal covers the engine. The property forms and the machines
that build on it get their reference and their vectors in the standard
first, and a Go design of their own after that.

## Motivation

The definition states what the engine does. Go has to settle how:

- **Where a typed value meets an untyped choice.** A generator returns a
  `T`, and the shrinker edits choices that have no Go type.
- **How a failure ends a case and not the test.** An assertion in a body
  reports through the case, and a failing aborting assertion has to end
  that case.
- **How the vectors call the engine.** A vector states a generator and
  a body as JSON, and pins the choices a case recorded. A black-box test
  of a public package can call neither.
- **Where the store is on disk, how workers run, and how a fuzzer's
  bytes become a case.**

## Detailed design

### Packages

```text
go.dokimi.dev/assert/prop                     ForAll, Fuzz, Case, Generator[T], the generators, the options
go.dokimi.dev/assert/internal/prop/choice     choice kinds, bounds, targets, sort keys, replay coercion
go.dokimi.dev/assert/internal/prop/random     the random source and every draw
go.dokimi.dev/assert/internal/prop/alphabet   the default alphabet
go.dokimi.dev/assert/internal/prop/token      the replay token
go.dokimi.dev/assert/internal/prop/store      the store's entries, file names, verdicts and directory
go.dokimi.dev/assert/internal/prop/coverage   the coverage test
go.dokimi.dev/assert/internal/prop/tree       the case tree
go.dokimi.dev/assert/internal/prop/engine     the case, providers, generators, runner, shrinker, explain
go.dokimi.dev/assert/internal/prop/pattern    the portable pattern subset and string-matching
go.dokimi.dev/assert/conformance              the vectors, beside the corpus runner
```

Each package imports only the packages above it in this list, and the
compiler's cycle check enforces that order. `pattern` imports `engine`,
because a pattern's nodes decode from a case, so `string-matching` is
built in `pattern` and not in `engine`.

The engine is internal for the reason the matcher is. Its logic exists
once, and `prop` is a surface of thin wrappers whose types and methods
`go doc` lists. `conformance` imports the engine directly, so the
vectors drive the code that every caller runs.

### The public surface

```go
package prop

// ForAll runs body against generated cases and stops the test with the
// smallest counterexample the shrink passes find.
func ForAll(tb assert.TB, contract string, body func(*Case), opts ...Option)

// Fuzz registers body as f's fuzz target. Each input's bytes decode into
// the choices of one case.
func Fuzz(f *testing.F, contract string, body func(*Case), opts ...Option)

// Case is one call of a body. It is a seat, so every assertion works in
// a body. Every method is safe for concurrent use.
type Case struct{ /* unexported */ }

var (
	_ assert.TB       = (*Case)(nil)
	_ assert.Reporter = (*Case)(nil)
	_ assert.Clocked  = (*Case)(nil)
)

func (c *Case) Helper()
func (c *Case) Fatalf(format string, args ...any)
func (c *Case) Errorf(format string, args ...any)
func (c *Case) Report(f assert.Failure, aborting bool)
func (c *Case) Clock() assert.Clock
func (c *Case) Assume(condition bool)
func (c *Case) Classify(label string)
func (c *Case) Note(message string)
func (c *Case) Rand() rand.Source // math/rand/v2
func (c *Case) Observe(fingerprint uint64)

// Draw returns a value of g and records it under label. Call Draw on the
// goroutine that runs the body: a draw from another goroutine records
// its choices in the order the scheduler gives, and a replay then
// decodes other values.
func (c *Case) Draw[T any](g Generator[T], label string) T

// Generator is a domain of T and how a case decodes one of its values.
type Generator[T any] struct{ /* unexported */ }

func (g Generator[T]) Map[U any](f func(T) U) Generator[U]
func (g Generator[T]) Filter(keep func(T) bool) Generator[T]
func (g Generator[T]) Bind[U any](f func(T) Generator[U]) Generator[U]

func Composite[T any](f func(*Case) T) Generator[T]

// Integral is every integer type, and Floating is float32 and float64,
// with the types defined over them.
type Integral interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}
type Floating interface{ ~float32 | ~float64 }

func Integer[T Integral](lo, hi T) Generator[T]
func Float[T Floating](lo, hi T, opts ...FloatOption) Generator[T]
func Boolean(opts ...BooleanOption) Generator[bool]
func Just[T any](value T) Generator[T]
func SampledFrom[T any](values ...T) Generator[T]
func OneOf[T any](gens ...Generator[T]) Generator[T]
func Optional[T any](of Generator[T]) Generator[*T]
func List[T any](of Generator[T], opts ...ListOption) Generator[[]T]
func Dict[K comparable, V any](keys Generator[K], values Generator[V], opts ...SizeOption) Generator[map[K]V]
func String(opts ...StringOption) Generator[string]
func Bytes(opts ...SizeOption) Generator[[]byte]
func Duration(lo, hi time.Duration) Generator[time.Duration]
func Permutation[T any](values ...T) Generator[[]T]
func StringMatching(pattern string) Generator[string]
func Recursive[T any](base Generator[T], extend func(self Generator[T]) Generator[T], opts ...RecursiveOption) Generator[T]

// SizeOption bounds the length of a list, a dict, a string or a byte
// string. A SizeOption is also a ListOption and a StringOption.
type SizeOption struct{ /* unexported */ }

func MinSize(n int) SizeOption
func MaxSize(n int) SizeOption

// ListOption configures List: a SizeOption, or Unique.
type ListOption interface{ /* unexported method */ }

// Unique makes List discard an element equal to an earlier one. Values
// compare as the definition compares them: by type and value, and floats
// by their bits.
func Unique() ListOption

// StringOption configures String: a SizeOption, or Alphabet.
type StringOption interface{ /* unexported method */ }

// Alphabet states the characters String chooses from, the simplest first.
func Alphabet(chars string) StringOption

type FloatOption struct{ /* unexported */ }

// AllowNaN lets Float return NaN.
func AllowNaN() FloatOption

type BooleanOption struct{ /* unexported */ }

// Odds makes Boolean return true with probability num/den.
func Odds(num, den uint64) BooleanOption

type RecursiveOption struct{ /* unexported */ }

// MaxLeaves bounds the values Recursive draws from its base; 100 by
// default.
func MaxLeaves(n int) RecursiveOption

// Option configures a run of ForAll or Fuzz.
type Option struct{ /* unexported */ }

func Cases(n int) Option
func Seed(s uint64) Option
func Replay(token string) Option
func Require(label string, share float64) Option
func Shrink(runs int) Option
func ShrinkTime(d time.Duration) Option
func MaxChoices(n int) Option
func Store(dir string) Option
func Explain(enabled bool) Option
func Workers(n int) Option
```

`Integer[uint64]` covers the whole unsigned range. `Float[float32]`
draws choices of width 32: the type argument states a float's width. The
definition leaves the form of a generator's parameters to each language.
Go states them as functional options, the form its run options take.

`Draw`, `Map` and `Bind` are generic methods, a feature of Go 1.27. A
property then reads `c.Draw(g, "label")` and `g.Map(f)`, as it does in
the other languages. A generic method never satisfies an interface.
Neither `Case` nor `Generator` needs to satisfy one.

A constructor panics when its arguments state no domain: `Integer(5,
1)`, a `SampledFrom` of nothing, a pattern outside the portable subset.
The definition requires such a generator to fail when it is built and
not when a case runs, and a constructor that returned an error would
make every property spend a line on a mistake in a literal.
`regexp.MustCompile` takes the same position.

`ShrinkTime` bounds a shrink on the platform clock, `assert.System{}`,
and not on the seat's clock, as the definition requires. A property under
`assert.Controlled` then still ends its shrink.

### Typed generators over untyped choices

`engine.Generator[T]` contains the generator's id and a function from a
case to a `T`. A case records choices. A choice has a kind, bounds and
a value, and no Go type:

```go
package choice

// Int is the value of an integer choice: any integer from -2^63 to
// 2^64 - 1. Its zero value is 0.
type Int struct{ /* unexported: a magnitude and a sign */ }

// Choice is one decision of a case.
type Choice struct {
	Kind Kind
	// Integer is the value of an integer choice. Float and Sequence
	// store the others.
	Integer  Int
	Float    float64
	Sequence []uint32
}
```

A replay token or a stored case can record −5 for a generator whose
bounds are now unsigned. An `Int` keeps the sign of −5 whatever bounds
requested it, so the replay coerces −5 to the target, as the definition
requires. The replay does not read the same bits as 2^64 − 5.

A sequence element is a `uint32`. Every sequence has fewer than 2^32
element values. `token.Decode` reads an element of 2^32 or more as
2^32 − 1, and a replay fits that element to 0 as it fits any element at
or above `k`. `Decode` checks the canonical form of each number on its
bytes, before it saturates the number. A token with a superfluous LEB128
byte is refused whatever the value of the number.

The type exists only at the edge where a generator turns choices into a
value. The shrinker, the case tree and the token never see a `T`. A
draw allocates for the record of the value a counterexample reports, and
for nothing else that a typed value needs.

`prop.Generator[T]` wraps `engine.Generator[T]`, and each method calls
the engine's. The conformance package builds `engine.Generator[any]`
values from a vector's generator spec, with `engine.Map` turning each
typed value into `any`. `Map` adds no span, so a vector records the
choices and the spans a typed generator records.

### A case is a seat

Each case has an `assert.Recorder` of its own, built with
`assert.NewRecorder().WithGoexit().WithClock(clock)`. `clock` is the
clock of the seat passed to `ForAll`. `Case` forwards `Helper`, `Report`
and `Clock` to the recorder, which already supplies what a case needs
from a seat:

- Every method is safe for concurrent use, so an assertion may report to
  the case from any goroutine, as the standard requires of every seat.
- `Report` keeps every record in call order. The case's failure is the
  first record in `Failures()`.
- Under `WithGoexit`, a fatal report ends the calling goroutine through
  `runtime.Goexit`.
- `Clock` returns the test's clock, so every case of a property runs
  under it.

`Case.Fatalf` and `Case.Errorf` report a record with no assertion
through the recorder's `Report`. The record's contract is the message,
and its `Where` is the innermost frame in the caller's code. That frame
is the first whose file ends in `_test.go` or whose function is outside
this module, so a test of this module attributes a failure to itself.
`Fatalf` reports the record as aborting and `Errorf` as recording. The
recorder's own `Fatalf` and `Errorf` keep no record for a plain message,
because the corpus runner reads a recorder with no record as an
assertion that reported none, so the case builds the record itself.

The draw state is the choices, spans and draws a case records, with
its labels, notes and fingerprints. It has a mutex of its own, so every
method of `Case` is safe for concurrent use.

| What the body does | What the case records | How the case ends |
|---|---|---|
| An aborting assertion fails | Its record | At once |
| A recording assertion fails | Its record | When the body returns |
| It calls `Fatalf` | A record with no assertion, at the innermost frame in the caller's code | At once |
| It calls `Errorf` | A record with no assertion, at the innermost frame in the caller's code | When the body returns |
| It panics | The panic value's type, at the innermost frame in the caller's code | At once |
| It calls `Assume(false)`, or a filter exhausts its attempts | A rejection | At once |
| A draw repeats a tested case, diverges, or passes `MaxChoices` | The signal | At once |

A failure's identity is its assertion, its file and its line, and for a
panic the value's type with the file and the line. A message is not part
of it.

The runner starts each case's body on a goroutine of its own, and a case
ends at once through `runtime.Goexit`. Deferred calls still run, and a
`recover` returns nil during a Goexit, so a body that recovers every
panic cannot swallow the end of its case. `testing` ends a test the same
way. `Fatalf` from another goroutine ends that goroutine only, and the
case fails when its body returns.

`Rand` returns a source whose every value is an integer choice over the
whole unsigned range.

### The record a failing run reports

`ForAll` reports one record with the assertion `prop-for-all`, the
caller's contract, and the ten detail fields of the definition. A field
that the outcome does not use is nil. The values are Go types. A test
reads them back from a `Recorder` without parsing a sentence:

```go
// Drawn is one value of a counterexample.
type Drawn struct {
	Label string
	Value any
	// Relevance states what the explain phase found for this draw.
	Relevance Relevance
	// NearestPassing is the nearest value that passes, for an integer
	// or a duration that matters, and nil otherwise.
	NearestPassing any
}

// Relevance is what the explain phase found for one draw.
type Relevance uint8

const (
	Untested      Relevance = 0 // untested
	AnyValueFails Relevance = 1 // any-value-fails
	ValueMatters  Relevance = 2 // value-matters
)
```

| Field | Go type |
|---|---|
| `outcome` | `Outcome` |
| `cases`, `rejected` | `int` |
| `seed`, `choices` | `string` |
| `counterexample` | `[]Drawn` |
| `failure` | `assert.Failure`, the failing case's own record |
| `others` | `[]Other` |
| `divergence` | `*Divergence` |
| `coverage` | `*Shortfall` |

Each enumeration is a `uint8` with a `String` method from stringer. The
line comment of each constant is its spelling in the definition.

### Workers

`Workers(n)` runs up to n cases at once, each on a goroutine of its own.
A prefix case starts once its random case ends, because it replays that
case's first choices. The runner reads the results in the order one
worker produces them and enters each into the case tree in that order.
A case that runs ahead keeps every choice it made, the attempts that a
filter removed from its record included, so the runner enters the walk
that the case would have made on one worker.
It cancels the cases it no longer needs once a case fails or enough have
passed. The shrinker evaluates up to n
candidates of a pass at once and accepts the first in the pass's order.
A run on n workers reports what a run on one reports. The vectors check
that with a setting of `workers: 4`.

### The store

A test's store is `testdata/prop/<test name>/` relative to the test's
directory, the directory that contains `testdata/golden`. The test name is
what the seat's `Name` method returns, with each subtest a directory of
its own. A seat with no `Name` method, such as a `Recorder`, has no
store unless `Store(dir)` states one, and `Store("")` turns the store
off. An entry is written with mode 0o644 in a directory of mode 0o755,
as a golden file is. A store that cannot be written is reported as a
note and does not fail the test.

A run reads every `.json` file of the directory and gives each the
definition's verdict. It replays the entries of its property oldest
first. A damaged file fails the test with the file's name, as a damaged
golden file does. A note on a passing run goes to the seat's `Logf`
when the seat has one. The run creates an entry's file only when no file
of that name exists, so it never overwrites an entry.

The identity's `file` is the base name of the failing frame's file. A
drawn value that no typed literal states is recorded as the literal of
its public form, and by its label alone when it has none:

| Value | Recorded as |
|---|---|
| A struct | A map of its exported fields, in declaration order |
| An `encoding.TextMarshaler` | A string, its text |
| A pointer | Its target, or null |
| A value of a defined type | Its underlying value |
| Any other value | Its label alone |

A value whose literal would nest past the format's 64 levels is recorded
by its label alone. So is a value of more than 65,536 parts, counting
each value as one part and each byte of a string as one more. The bound
keeps an entry reviewable, and it ends the walk of a value that contains
itself.

### The fuzz bridge

```go
func FuzzRoundTrip(f *testing.F) {
	prop.Fuzz(f, "decoding undoes encoding", func(c *prop.Case) {
		v := c.Draw(prop.Bytes(), "input")
		got, err := Decode(Encode(v))
		assert.NoError(c, err, "decoding succeeds")
		assert.Equal(c, got, v, "decoding returns the encoded bytes")
	})
}
```

`Fuzz` calls `f.Fuzz` with a function of `(*testing.T, []byte)`. Each
input's bytes decode into one case by the definition's bridge rules. A
failing case is shrunk, written to the store, and reported through the
input's `*testing.T` with its replay token. `go test` without `-fuzz`
runs the seed corpus and the stored cases.

### Conformance

`conformance/vectors.go` reads `spec/corpus/prop/*.json` from the
embedded definition. `//go:embed spec/corpus` already embeds the
subdirectory, and the corpus runner's glob of `spec/corpus/*.json` does
not read it. Each kind of vector has its own runner:

| Kind | What the runner does |
|---|---|
| `decoding` | Builds the generator, replays the stated choices, and compares the recorded choices, the value and the rejection |
| `generation` | Draws the first cases of the seed, and compares each case's choices and value |
| `shrinking` | Runs the property, and compares the outcome, the cases, the minimal value, its choices and token, and the runs |
| `coverage` | Compares the verdict |
| `bridge` | Decodes the bytes, and compares the choices and the value |
| `token` | Encodes the choices or decodes the token, and compares the result or the refusal |
| `behaviour` | Runs the body under the settings with a `Recorder`, and compares every detail field |
| `store` | Writes the entry and its name and compares both, or reads the text and compares the verdict and the choices |

A body spec becomes a Go body: a `fails` entry reports a record whose
assertion is the entry's identity, so the identity a vector names is the
identity the engine keeps. A typed literal decodes into the value the Go
generator returns, and a map compares without its order, because a Go
map has none. Every other comparison is exact. `conformance/literal.go`
gains the `bytes`, `items` and `entries` forms and the large integer
strings.

### Testing

- Every source file has a black-box test file beside it.
- The vectors pin every algorithm the definition fixes. A unit test pins
  each contract that no vector states, such as a goroutine that `Fatalf`
  ends.
- Every exported function and method has a benchmark under
  `bench.Start(b).MaxAllocs(n)`. The random source, the draws, the sort
  keys and the token encoder state 0. Every other ceiling is the
  measured count.
- Every package has 100% statement coverage and 100% mutation
  coverage under gremlins.

### The overlay's limit on dict

Go has `testing.F`, `testdata`, goroutines and the whole unsigned range,
so the overlay declines nothing of the property surface. It declares
one limit, on `prop.dict`. A Go map with float keys stores the two zeros
under one key and every NaN under a key of its own. The definition keeps
the two zeros apart and treats every NaN as one value. The draw follows
the definition, and the map then stores what Go's equality allows.

## Alternatives considered

### A. The engine in the public package

Put the engine in `prop` and test it from `prop_test`.

**Why not:** a vector pins the choices a case recorded and decodes a
generator built from JSON. A black-box test can call neither without an
exported hook. An exported hook puts the engine's internals on the
public surface, where every later change to them breaks a caller.

### B. An untyped engine with typed wrappers

Generators return `any`, and `Case.Draw` asserts the type.

**Why not:** every draw boxes its value, and every `List[T]` converts a
`[]any` into a `[]T` with an assertion per element. The vectors would
drive the untyped path, and callers the conversions, so the code the
vectors check would not be the code callers run.

### C. A panic to end a case

The runner calls the body on its own goroutine and recovers a sentinel
panic.

**Why not:** a test of a server that recovers panics has a body that
recovers every panic. Such a body swallows the sentinel and runs on
past the end of its case. The definition requires a body to let the
signal pass, and Goexit ends the case whatever the body recovers. The
cost is a goroutine per case.

### D. Generic type aliases for the surface

`type Generator[T any] = engine.Generator[T]`.

**Why not:** `go doc` lists no methods for an alias of a type in an
internal package. The assertion chain met the same problem, and the
public package declares its own types for the same reason.

### E. A seat of the case's own

Keep the case's failures in `Case` itself, and end the case with its own
call to `runtime.Goexit`.

**Why not:** `assert.Recorder` is already a TB, a Reporter and Clocked.
It is safe for concurrent use, keeps every record in call order, and
under `WithGoexit` ends the calling goroutine on a fatal failure. A
second seat would repeat that code. The two copies could then drift
apart.

### F. Draw through the generator

Draw with `g.Draw(c, label)`, and make `Map` and `Bind` package
functions. That form works in every Go version with generics.

**Why not:** every other language draws through the case and maps
through the generator. The naming rule allows a different spelling only
where a language requires one. Go 1.27's generic methods remove that
requirement. This module already requires Go 1.27.

### G. A record in the Recorder for every plain message

Make `Recorder.Fatalf` and `Recorder.Errorf` keep a record at the
caller's frame, so a case needs no `Fatalf` of its own.

**Why not:** the corpus runner reads a recorder with no record as an
assertion that reported none. A recorder that kept a record for every
plain message would hide an assertion that calls `Fatalf` where it
should report its record. The recorders of the Go, Python, Rust,
TypeScript and Java implementations keep no record for a plain message,
for that reason.

## Drawbacks

- **The engine is the largest code in the module.** The reference's
  engine is 4,300 lines of Python in 17 modules. A Go translation with
  explicit types and error paths is likely to run to 6,000 lines in 8
  internal packages, with the tests on top.
- **Each public function is a wrapper.** The surface has 44 functions
  and methods that call the engine's, each a few lines with a docblock
  of its own: 35 functions, 3 methods of `Generator` and 6 of `Case`.
- **A case costs a goroutine and a recorder.** Goexit needs the
  goroutine. A property of 100 cases starts 100 goroutines and builds
  100 recorders, and the benchmarks measure what that costs before any
  ceiling is stated.
- **A float-keyed `Dict` returns a map that the definition's value does
  not match.** The overlay states the limit.
- **A change to the definition is two changes.** The reference changes
  in the standard and the engine changes here. The vectors fail until
  both agree.

## Open questions

None.

## Unresolved and future work

- The property forms and the inputs derived from types are not proposed
  here.
- Machines, simulation and campaigns are not proposed here.

## References

| What | Where |
|---|---|
| The definition and its property engine, version 1.2.0 | <https://github.com/dokimasia/assert-spec> |
| `runtime.Goexit` | <https://pkg.go.dev/runtime#Goexit> |
| `testing.F.Fuzz` | <https://pkg.go.dev/testing#F.Fuzz> |
| `math/rand/v2.Source` | <https://pkg.go.dev/math/rand/v2#Source> |
