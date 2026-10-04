---
rfc: 0004
title: Property forms and derived inputs in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-02
updated: 2026-10-03
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0004: Property forms and derived inputs in Go

## Summary

Definition 2.2.0 adds RFC-0011 of the standard: 38 property forms, the
27 shapes that derive a generator from a type, the inverse of every
generator, registrations, shape files, and the `Example` and `Draws`
options. This RFC fixes how Go reads a type into a shape, the Go value
that each shape decodes to, the signatures of the forms and the nine
helpers, how one option list takes both the engine's options and the
relaxations, and the tests that check each part against the definition.

The module's dependencies do not change. Go 1.27's standard library has
`uuid`, `net/netip`, `math/big` and the time-zone database.

## Motivation

The definition fixes the shapes, their generators and their inverses,
and pins each with vectors. Go decides the rest:

- Which Go types read as which shape, and how a struct tag states a
  constraint or chooses between two shapes of one type.
- What a record's field is called. A Go field is exported, so its name
  starts with a capital letter, and the fixture vectors name fields in
  lowercase.
- Which Go value each shape decodes to in `OfShape`, where Go has no
  type for a shape.
- How one variadic list takes `prop.Cases(500)` and
  `assert.EquateNaNs()`, which are values of two Go types.
- Where the zone table is at run time. The vendored definition is test
  data of the conformance package, and a library cannot embed a file
  outside its own directory.

## Detailed design

### Components

| Component | Responsibility |
|---|---|
| `internal/prop/literal` | The typed-literal codec: `Decode`, moved from `conformance`, `Encode`, moved from `internal/prop/store`, and the types `Record`, `Pairs` and `Variant` of the neutral values |
| `internal/prop/zone` | The zone list and every offset change of its zones, embedded from a copy of the definition's `zones.json` |
| `internal/prop/engine` | The inverse of every generator, `MapBack`, the examples a run tries first, and the provider of a `Draws` case |
| `internal/prop/shape` | The 27 shapes: reading a shape document with its definitions, the recursion budget, each shape's generator of neutral values, and its inverse |
| `prop` | The type reader, the registry, `Of`, `Register`, `RegisterValues`, `RegisterVariants`, `Using`, `Example`, `Draws`, `ShapeOf`, `OfShape`, the 38 forms, and the types `FormOption`, `Variant` and `WallTime` |
| `internal/matcher` | `FormSeal`, the type that seals `FormOption` to this module, and the method that makes a relaxation a `FormOption` |
| `conformance` | The runners of the shapes, inverse, fixtures, draws and forms vectors, the 34 fixture types, the ten new subject kinds, and the pins of the new names |
| `conformance/spec/` | Definition 2.2.0, vendored with `make spec-sync`, which also copies `zones.json` to `internal/prop/zone` |

The packages keep the order of RFC-0002. `literal` follows `pattern`,
and `store` imports it. `zone` follows `tree`. `shape` follows
`matching`, because a string shape with a pattern decodes through
string-matching. `prop` imports `shape`, and `conformance` imports both.

### Neutral values

`internal/prop/shape` decodes each shape to the value that the
definition's reference decodes. This RFC calls it a neutral value:

| Shape | Neutral value |
|---|---|
| `bool` | `bool` |
| `int` of a width up to 64 | `int64`, or `uint64` when the shape is unsigned |
| `int` of width 128 | `*big.Int` |
| `float` | `float64`, or `float32` at width 32 |
| `char` | `string` of the one character |
| `string` | `string` |
| `bytes`, `uuid`, `ip-address` | `[]byte` |
| `list`, `fixed-list`, `set` | `[]any` |
| `map` | `literal.Pairs`: the entries in the order the case generated them |
| `optional` | `nil`, or the value |
| `record` | `literal.Record`: the fields in declaration order |
| `enum` | `literal.Variant`: the name, and the payload when the variant has one |
| `literal` | The value that the typed literal states |
| `decimal`, `date`, `time-of-day`, `duration`, `offset` | `int64`: the unscaled value, the days, the units, the units or the seconds |
| `instant`, `local-date-time`, `zoned-date-time`, `wall-time` | `literal.Record` of the parts that the definition states, whose names the shape package exports |
| `zone` | `string`: the zone's name |

A vector states its values in this form, so the conformance package
compares a neutral value with a typed literal directly. The type reader
and `OfShape` convert a neutral value to a Go value, and the inverse
converts a Go value back to a neutral one. `Drawn.Neutral` returns the
neutral value of a draw, and a store entry records it, so an entry of a
derived input is a typed literal that every language reads.

### Reading a type

`prop` reads a Go type through `reflect`. The table states the shape of
each type that it reads:

| Go type | Shape |
|---|---|
| `bool` | `bool` |
| `int8` to `int64`, `uint8` to `uint64` | `int` of the type's width, signed as the type is |
| `int`, `uint` | `int` of the platform's word: 64 bits, or 32 on a 32-bit platform |
| `float32`, `float64` | `float` of the type's width |
| `string` | `string` |
| `rune` with the tag `char` | `char`. Without the tag, a `rune` is an `int32` |
| `[]byte`, `[N]byte` | `bytes`, with `min_size` and `max_size` N for an array. A slice or an array of another type over `uint8` reads as a `list` or a `fixed-list` of integers |
| `uuid.UUID` | `uuid` |
| `netip.Addr` | `ip-address` |
| `*big.Int` | `int` of width 128, signed |
| `*big.Rat` | `decimal`. The tag states its `scale` |
| `time.Time` | `instant` at nanoseconds. The tag `date`, `local-date-time` or `zoned-date-time` reads it as that shape |
| `time.Duration` | `duration` at nanoseconds. The tag `time-of-day` reads it as that shape |
| An integer type with the tag `offset` | `offset`, in seconds east of UTC |
| `*time.Location` | `zone` |
| `WallTime` | `wall-time` at nanoseconds |
| `[]T`, `[N]T` | `list`, and `fixed-list` of size N |
| `map[K]struct{}` | `set` of K |
| `map[K]V` | `map` from K to V |
| `*T` | `optional` of T |
| A struct | `record` of its exported fields, in declaration order |
| A type that `RegisterValues` registered | `literal` over the registered values |
| An interface that `RegisterVariants` registered | `enum` over the registered types |

A field's name in the record is the name that its `json` tag states, and
its Go name when the tag states none. Go already names a field for every
other language through that tag, and the fixture types state `count`
where Go declares `Count`. The tag `prop:"-"` leaves a field out of the
record. An unexported field is never read, and both kinds keep their
zero value.

A defined type over a basic kind, such as `type Status int`, reads as
its underlying type over the whole range, unless `RegisterValues`
registered it. A named type that refers to itself becomes a definition
named by its package path and its name, such as
`example.com/shop.Tree`, and each place it occurs becomes a `ref`. The
reader refuses every other type: an interface that `RegisterVariants`
did not register, a channel, a function, a complex number, `uintptr` and
`unsafe.Pointer`. The error names the path of the field, such as
`Order.Lines[].Note`.

A unit follows the type, as the definition states. A `time.Time` and a
`time.Duration` are at nanoseconds, and the tag `unit` states a coarser
one. The reader reads each type once per process and keeps its shape and
its converters in a `sync.Map`.

### Tags

A tag is `prop:"key=value,key"`. Each key states one constraint of the
definition's table or chooses a shape:

| Key | Applies to | Effect |
|---|---|---|
| `min`, `max` | numbers, `decimal`, and the date and time shapes | Bounds the value, in the form a shape file states the bound |
| `min_size`, `max_size` | `string`, `bytes`, `list`, `set`, `map` | Bounds the size |
| `pattern`, `alphabet` | `string`, and `alphabet` for `char` | The rest of the tag after `=` is the value, so each is the last key |
| `allow_nan`, `allow_infinity` | `float` | Admits NaN or the infinities |
| `unit` | `time.Time`, `time.Duration`, `WallTime` | `s`, `ms`, `us` or `ns` |
| `scale` | `*big.Rat` | The digits after the point |
| `version` | `netip.Addr` | 4 or 6 |
| `char`, `date`, `local-date-time`, `zoned-date-time`, `time-of-day`, `offset` | the types in the reading table | Reads the type as the named shape |
| `-` | any field | Leaves the field out |

The reader copies each value into the shape, so the shape package parses
every bound once, as it parses a shape file. It gives each key to the
field's own shape when that shape has a parameter of the key's name, and
any other key to the shape inside an `optional`, a `list`, a `fixed-list`
or a `set`, so `prop:"max_size=3,min=1"` on a `[]int32` bounds the list
and its elements. A key that the table does not state, and a key that no
part of the field's type takes, fail the read. The error names the field
and the key.

### Registrations

```go
func Of[T any]() Generator[T]
func Register[T any](g Generator[T])
func RegisterValues[T any](values ...T)
func RegisterVariants[I any](variants ...I)
func Using[T any](g Generator[T]) FormOption
```

A registration applies to the whole test process, as the definition
states. The registry is one map guarded by a mutex. The first run of
`ForAll`, `Fuzz` or a form closes it, and a registration after that
panics. A second registration of one type panics too, because two
packages that register one type in `init` would otherwise depend on the
order in which Go initialises them. A registration of a type that a read
has already looked up panics as well, because the shape that the read
returned does not state the registration. A panic is the position that
RFC-0002 takes for a constructor whose arguments state no domain. `Of`
panics for the same reason for a type that the reader refuses, and the
panic names the field. A form reports the reader's error as a failure of
its test instead, because it has a seat.

`RegisterVariants` names each variant by its type's name without the
package path, so the variants of `Event` are `Created`, `Refunded` and
`Cancelled`. Two variants of one name panic. A generator from `Using`
applies to each place its type occurs in the inputs of the form that the
option is passed to. A form takes a generator from `Using` first, then
from the registry, then from the type's shape.

### Shape files

```go
func ShapeOf[T any]() (string, error)
func OfShape(text string) (Generator[any], error)
```

`ShapeOf` writes the shape of T with its definitions and the field
`source`, such as `{"language": "go", "type": "example.com/shop.Order"}`.
Keys are sorted and indented by two spaces, with a final newline, so a
golden file of a shape changes only when the type does. It returns an
error for a type that the reader refuses, and for a type that contains a
type whose registered generator has no shape.

`OfShape` returns an error for a document that the definition's rules
refuse, because its text comes from a file at run time and not from a
literal. It decodes each shape to the Go value of this table:

| Shape | Go value |
|---|---|
| `int` | `int8` to `int64` or `uint8` to `uint64` by its width, and `*big.Int` at width 128 |
| `float` | `float32` or `float64` by its width |
| `char` | `rune` |
| `bytes` | `[]byte` |
| `list`, `fixed-list`, `set` | `[]any` |
| `map` | `map[any]any`. A key shape whose values Go cannot compare, such as a list, fails the read |
| `optional` | `nil`, or the value |
| `record` | `map[string]any`, as the definition states |
| `enum` | `Variant` |
| `uuid`, `ip-address`, `decimal` | `uuid.UUID`, `netip.Addr`, `*big.Rat` |
| `instant`, `date`, `local-date-time` | `time.Time` in UTC, and a date at midnight. The inverse takes an instant in any location, and refuses a date or a local date and time outside UTC |
| `time-of-day`, `duration` | `time.Duration` |
| `offset` | `int`, in seconds east of UTC |
| `zone` | `*time.Location` |
| `zoned-date-time` | `time.Time` in its zone |
| `wall-time` | `WallTime` |

```go
// Variant is a value of an enum shape: the variant's name, and its
// payload when the variant has one.
type Variant struct {
	Name       string
	Payload    any
	HasPayload bool
}

// WallTime is a value of the wall-time shape: a wall time in UTC's
// fields, and the zone that the code under test resolves it in.
type WallTime struct {
	Local time.Time
	Zone  *time.Location
}
```

`HasPayload` keeps a variant without a payload apart from one whose
optional payload is absent, as the definition requires.

### From a value back to choices

Each generator of the engine gains its inverse, by the order that the
definition fixes:

- A `one-of` takes its first alternative that produces the value.
- A `sampled-from` takes the first equal value.
- A `permutation` takes the smallest index at each swap.
- A `string-matching` pattern takes the match that a backtracking engine
  finds first.
- An optional is absent first, and a recursive value takes its base
  first.
- A set's elements and a map's entries take the shortlex order of their
  own choices.
- A zoned value runs back through the branch of the whole range.

`Filter` inverts through its source and refuses a value that its
predicate rejects. `Map`, `Bind` and `Composite` have no inverse. The
method `g.MapBack(f, back)` maps g and states the inverse of `f`. Every
shape generator built from an integer keeps its inverse and the explain
phase's step that way. `NewInvertible` builds a generator from its decode
function and its inverse. `Erase` keeps the inverse when it erases a
generator's type. `engine.Invert(g, value)` returns the choices, or an
error that names the part of the value that no choice produces. The shape
package's `ReadWith` reads a shape file whose refs name generators
without a shape, such as a registered one. `Of` reads a type that
contains a registered type that way, with the registered generator at
each place of that type.

### Example and Draws

```go
func Example[T any](values ...T) FormOption
func Draws(entries string) Option
```

`Example` states one value for each argument that a form generates: one
for a form over a function, two for `Commutative` and three for
`Associative`. A form computes each example's choices when it is built,
and fails the test there for a wrong number of values, a value of
another type than the form's input, or a value that its generator cannot
produce. A run tries its examples first, before the stored cases. A
failing example shrinks like any other case.

`Draws` takes a JSON array of entries, each a label and a typed literal,
as a store entry records its counterexample. The run's first case reads
them through a provider of its own. Each draw takes the next entry,
checks the label, and computes the choices with its own generator's
inverse. A label that differs, or a value that the draw's generator
cannot produce, fails the test before any other case runs, and the
error names the label. The engine ends such a run with `Result.Refused`,
which the test's failure reports. A draw past the last entry takes its
target. `Draws` is an `Option`. It applies to `ForAll` and to a form
alike, and `Fuzz` does not run a case of entries.

### The forms

```go
func Equal[T, U any](tb assert.TB, got, want func(T) U, msg string, opts ...FormOption)
func NotEqual[T, U any](tb assert.TB, got, want func(T) U, msg string, opts ...FormOption)
func True[T any](tb assert.TB, cond func(T) bool, msg string, opts ...FormOption)
func False[T any](tb assert.TB, cond func(T) bool, msg string, opts ...FormOption)
func Nil[T, U any](tb assert.TB, got func(T) U, msg string, opts ...FormOption)
func NotNil[T, U any](tb assert.TB, got func(T) U, msg string, opts ...FormOption)
func Length[T, U any](tb assert.TB, got func(T) U, want int, msg string, opts ...FormOption)
func Empty[T, U any](tb assert.TB, got func(T) U, msg string, opts ...FormOption)
func NotEmpty[T, U any](tb assert.TB, got func(T) U, msg string, opts ...FormOption)
func Contains[T, U any](tb assert.TB, got func(T) U, needle any, msg string, opts ...FormOption)
func NotContains[T, U any](tb assert.TB, got func(T) U, needle any, msg string, opts ...FormOption)
func ContainsInOrder[T, U any](tb assert.TB, got func(T) U, needles []string, msg string, opts ...FormOption)
func IsPermutation[T, E any](tb assert.TB, got, want func(T) []E, msg string, opts ...FormOption)
func HasPrefix[T, U any](tb assert.TB, got func(T) U, prefix, msg string, opts ...FormOption)
func HasSuffix[T, U any](tb assert.TB, got func(T) U, suffix, msg string, opts ...FormOption)
func Matches[T, U any](tb assert.TB, got func(T) U, pattern, msg string, opts ...FormOption)
func CloseTo[T, U any](tb assert.TB, got func(T) U, want, tolerance float64, msg string, opts ...FormOption)
func InRange[T, U any](tb assert.TB, got func(T) U, low, high float64, msg string, opts ...FormOption)
func Pairwise[T, E any](tb assert.TB, got func(T) []E, pred func(earlier, later E) bool, msg string,
	opts ...FormOption)
func NoError[T any](tb assert.TB, fn func(T) error, msg string, opts ...FormOption)
func HasError[T any](tb assert.TB, fn func(T) error, msg string, opts ...FormOption)
func ErrorIs[T any](tb assert.TB, fn func(T) error, target error, msg string, opts ...FormOption)
func ErrorIsNot[T any](tb assert.TB, fn func(T) error, target error, msg string, opts ...FormOption)
func ErrorAs[E, T any](tb assert.TB, fn func(T) error, msg string, opts ...FormOption)
func Panics[T any](tb assert.TB, fn func(T), msg string, opts ...FormOption)
func NotPanics[T any](tb assert.TB, fn func(T), msg string, opts ...FormOption)
func Pure[T, S any](tb assert.TB, observe func() S, fn func(T), msg string, opts ...FormOption)
func NotPure[T, S any](tb assert.TB, observe func() S, fn func(T), msg string, opts ...FormOption)
func NilContextSafe[T any](tb assert.TB, fn func(ctx context.Context, in T) error, msg string,
	opts ...FormOption)
func HonoursCancellation[T any](tb assert.TB, fn func(ctx context.Context, in T) error, msg string,
	opts ...FormOption)
func HonoursDeadline[T any](tb assert.TB, fn func(ctx context.Context, in T) error, msg string,
	opts ...FormOption)
func MaxAllocs[T any](tb assert.TB, fn func(T), ceiling uint64, msg string, opts ...FormOption)
func Idempotent[I, S any](tb assert.TB, call func(I) error, observe func() S, msg string, opts ...FormOption)
func Accumulates[I any](tb assert.TB, call func(I) error, observe func() int, msg string, opts ...FormOption)
func Deterministic[I, O any](tb assert.TB, call func(I) (O, error), msg string, opts ...FormOption)
func Commutative[T, R any](tb assert.TB, combine func(a, b T) R, msg string, opts ...FormOption)
func Associative[T any](tb assert.TB, combine func(a, b T) T, msg string, opts ...FormOption)
func RoundTrip[I, E any](tb assert.TB, forward func(I) (E, error), inverse func(E) (I, error), msg string,
	opts ...FormOption)
```

Each form keeps its assertion's arguments after the function or the
callable, as the definition's rule states. Go infers every type
parameter from the arguments except the error type of `ErrorAs`, which
comes first. A caller states it as for `assert.ErrorAs`:
`prop.ErrorAs[*fs.PathError](t, open, msg)`. The completeness gate counts
that type parameter, because no parameter's type names it.

A form runs as `ForAll` runs, through the function that runs both, with
a body of its own. The body draws each
generated argument once, labelled `input`, or `a`, `b` and `c` for the
relations, and calls the assertion from `internal/matcher` with the case
as its seat and in the aborting mode. The run reports one record of the
form's own id, such as `prop-equal`, with the detail of `prop-for-all`.
Its `failure` is the record that the assertion reported for the minimal
case. A form whose input type has neither a generator nor a shape fails
the test before any case runs, and the error names the field.

A form takes the relaxations that its assertion accepts. A relaxation
passed to a form whose assertion accepts none fails the test before any
case runs. `MaxAllocs` runs one case at a time whatever `Workers` states,
because the count covers the whole process. It checks no ceiling in the
builds where `assert.MaxAllocs` checks none.

### One option list

```go
// FormOption configures a property form: an Option of the run, a
// relaxation of the form's assertion, or what Using and Example return.
type FormOption interface {
	FormOption(matcher.FormSeal)
}
```

The method's parameter type is in an internal package, so no type
outside this module implements `FormOption`. `Option` implements it, the
relaxation type of `assert` implements it through a method on
`matcher.Option`, and so do the values that `Using` and `Example` return.
The call that RFC-0011 writes then compiles as written:

```go
prop.Equal(t, decode, reference, "decode agrees with the reference",
	prop.Cases(500), assert.EquateNaNs())
```

`ForAll` and `Fuzz` keep `opts ...Option`, so a relaxation, `Using` and
`Example` cannot reach a run that has no form, and the compiler says so.

### The zone table

`internal/prop/zone` embeds `zones.json` with `//go:embed`. `make
spec-sync` copies the file from the vendored definition, and a test
compares the SHA-256 of the embedded bytes with the digest that the
vendored manifest states for `spec/zones.json`. A sync that left the copy
behind fails that test.

Go resolves a zone's name with `time.LoadLocation`, which reads the
database that `ZONEINFO` names, then the system's, then the copy in the Go
installation: tzdata 2026c in Go 1.27.1. `prop` does not import
`time/tzdata`, because Go's documentation reserves that decision for a
program's main package. The database decides the wall time that a
`time.Time` shows, and not the generated instant or zone. The Go overlay
declares the release for that reason.

### Conformance

| Kind | What the runner does |
|---|---|
| `shapes` | Reads the shape, decodes the first cases of the seed, and compares each case's choices and neutral value |
| `inverse` | Runs the shape or the generator back from the value, and compares the choices or the refusal |
| `fixtures` | Calls `ShapeOf` on the Go fixture type, and compares the shape with the stated one, without `source` and with each definition renamed by the order in which the shape first refers to it |
| `draws` | Runs the stated draws under a `Draws` case of the stated entries, and compares the choices, the values or the error |
| `forms` | Builds the subjects, runs the form over `OfShape` of the stated shape with `Using` and `Seed`, and compares the record's detail and its failure. A passing run reports no record, so the runner compares its counts through the engine's run of a body that draws what the form generates |

The fixture types are 34 Go declarations in `conformance/fixture.go`,
each with the `json` tags that name its fields. The package's `init`
registers the values of `status` and the variants of `payment`. The
fixture `colour` is an array of a type over `uint8`. An array of bytes
would read as `bytes`. Go names a definition by its type's package path
and a vector by the fixture's name, so the comparison renames each
definition.

The forms runner calls each form with `any` as its input type. Its
subjects take the values that `OfShape` decodes. Go's equality does not
equate an `int32` with an `int64`, so an integer that a subject makes,
and the needle of `prop-contains` and `prop-not-contains`, take the Go
type of the input's integers. The type of the subject's own failure is
unexported, and `prop-err-as` takes `error` as its type. A form's failure
compares its assertion and the fields that the vector states. The runners
read the `record` and `variant` forms of a typed literal through
`literal.Decode`.

`TestSurfaceTable` pins the nine helpers, and the completeness gate
checks each form's name and arity on the prop surface. The overlay gains
nothing for the forms: `prop-max-allocs` and the width of an `int` are
limits that definition 2.2.0 already declares.

### Testing

- The 209 vectors of definition 2.2.0 pin the shapes, the inverses, the
  fixture types, the draws and the forms.
- A unit test pins each rule that no vector states:
  - The reading of each Go type in the table, each tag key, and each
    refusal with its field path.
  - A registration after the first run, a second registration of one
    type, and a registration of a type that a read has looked up, each a
    panic.
  - `Example` with a wrong number of values, a value of another type, and
    a value outside the domain.
  - A relaxation passed to a form whose assertion accepts none.
  - `MaxAllocs` under `Workers(4)`.
  - The text of `ShapeOf` for a struct, a recursive type and a registered
    enum, against golden files.
  - The Go value of each row of the `OfShape` table.
  - Each refusal of the runners of the five new vector kinds, and each
    output of theirs that differs from the run, through a vector that the
    test builds.
- A property pins the inverse against the reader: every value that
  `Of[T]` generates for the test types, which cover each shape that the
  reader reads, runs back to choices that replay to an equal value.
- Every exported function has a benchmark under `bench.Start(b).MaxAllocs(n)`,
  as RFC-0002 requires.
- Every package keeps 100% statement coverage. The mutation stage of
  `ergon check` keeps 100% for the matcher and the conformance package.
  gremlins measures the engine's packages and `prop` on demand.

## Alternatives considered

### A. A wrapper for relaxations

`prop.Relax(assert.EquateNaNs())` turns a relaxation into an `Option`,
and every form takes `opts ...Option`.

**Why not:** the call that RFC-0011 writes would not compile. `Using` and
`Example` would be `Option` values that `ForAll` accepts and ignores, and
only a run-time check could refuse them.

### B. Field names from the Go names

A record's field is `Count`, or `count` after a rule that lowers the
first letter.

**Why not:** `Count` makes the shape of one type differ between Go and
every other language, so a shape file from Go matches no other reader. A
rule that lowers letters turns `ID` into `iD`. The `json` tag already
states the name that Go uses outside the program.

### C. Skips for the date and time fixtures

Go declares no type for `date`, `time-of-day`, `local-date-time`,
`offset` and `zoned-date-time`, and skips their five fixtures.

**Why not:** a Go caller who stores a date in a `time.Time` could not
derive dates. A tag that chooses the shape costs one key, as `char` does
for a `rune`.

### D. Locations built from the zone table

Build each `*time.Location` from the definition's offset changes with
`time.LoadLocationFromTZData`, so every platform resolves the same
offsets.

**Why not:** the table covers 1900 to 2100 and states no abbreviation. The
code under test resolves wall times with the platform's database in any
case, and a property should see the times that the code sees.

### E. An `OfShape` that panics

`OfShape(text) Generator[any]` panics on a document the rules refuse, as
a generator's constructor does.

**Why not:** a constructor's arguments are literals in a test's source.
A shape file is read at run time and copied between repositories.
`regexp.Compile` returns an error for the same reason.

### F. A generated reader

A `go generate` step writes each type's shape and converters. No
reflection runs.

**Why not:** RFC-0011 reads Go types through `reflect`. A generated file
is a second copy of the type and drifts from it. The reader reads each
type once per process, so reflection costs one read per type.

## Drawbacks

- **The surface grows by 38 forms, 9 helpers and 3 types**, each a
  wrapper with a docblock and a benchmark of its own.
- **`FormOption` has an exported method that no caller calls.** `go doc`
  lists it on `Option`.
- **A tag is checked when a property reads its type, and not at compile
  time.** A misspelt key fails the first run that reads the field.
- **The `json` tag names a record's field.** Renaming a JSON field renames
  the shape's field, which changes the shape file.
- **`OfShape` refuses a map whose key shape Go cannot compare**, such as a
  map keyed by lists that a Java type reads as. The Go overlay declares
  it.
- **The zone's wall time depends on the platform's database.** The
  overlay declares the release.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| The definition, version 2.2.0, and its RFC-0011 | <https://github.com/dokimasia/assert-spec> |
| `reflect` | <https://pkg.go.dev/reflect> |
| `uuid.UUID` | <https://pkg.go.dev/uuid> |
| `net/netip.Addr` | <https://pkg.go.dev/net/netip> |
| `time.LoadLocation`, and `time/tzdata`'s advice to libraries | <https://pkg.go.dev/time#LoadLocation> |
| `encoding/json` struct tags | <https://pkg.go.dev/encoding/json#Marshal> |
