---
rfc: 0008
title: One equality for every comparison
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-04
updated: 2026-10-04
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0008: One equality for every comparison

## Summary

The definition states two equalities:

- `equal` compares structurally. A nil collection differs from an empty
  one, NaN differs from itself, -0 equals +0, a cycle stops, and two
  functions are equal when they are the same function.
- The property engine compares two values by type and value, and two
  floats by their bits. -0 differs from +0, and every NaN is one value.

The module implements them in four places, and the four disagree:

- go-cmp, under the options of `matcher.Options`, decides `Equal`,
  `NotEqual`, `Pure`, `Permutation`, `Contains` on a list, and the
  relation family.
- `Contains` looks a map key up with `reflect.Value.MapIndex`.
- `prop` matches a Go value of a literal shape with `reflect.DeepEqual`,
  and compares a stored value with a replayed one the same way.
- `engine.canonicalKey` decides the property engine's identity.

This RFC gives each equality one implementation:

- A new package, `internal/equality`, implements `equal`. Every
  assertion that compares as `equal` compares calls it, with its
  relaxations built once per call. It calls no method of a value,
  compares a map key as it compares any value, and compares a map with a
  NaN key without a panic. Its `Diff` finds each difference of two
  unequal values under the same rules, and the writer renders the
  differences as the text of a failure.
- `engine.SameValue`, exported, implements the property engine's
  identity. The literal inverse and the store's comparison of recorded
  values call it. Its canonical key writes a pointer, a map or a slice
  met inside itself as a reference back to it.

go-cmp leaves the module, which then depends on the standard library
alone. The checks of `conformance` and of `internal/matchertest`, which
must not use the comparison that they check, compare with code of their
own.

Go cannot tell two closures of one function literal apart without unsafe
access. A function compares by its code pointer, and the Go overlay
declares that limit.

The RFC replaces the "Comparison rules" of RFC-0001, and amends the
dependency rule of `docs/standards/code.md`, the rule on `recover` of
`docs/standards/errors.md`, and the mutation and verdict rules of
`docs/standards/testing.md`.

## Motivation

The review of 9420a79 against definition 2.3.0 found five defects in
the comparisons. Probes against the working tree of 2026-10-04 reproduce
each of them:

| Finding | Call | Result | What the definition requires |
|---|---|---|---|
| 5 | `Contains(map[*node]int{{N: 1}: 1}, &node{N: 1})` | Fails | Passes: a key compares as `equal` compares, and the two keys are equal |
| 5 | `Contains(map[float64]int{NaN: 1}, NaN, EquateNaNs())` | Fails | Passes |
| 5 | `Contains(map[any]int{nil: 1}, nil)` | Fails | Passes |
| 5 | `Contains(map[any]int{1: 1}, []int{1})` | Panics: hash of unhashable type `[]int` | Fails |
| 6 | `Equal(always{1}, always{2})`, where `always.Equal` returns true | Passes | Fails: the fields differ |
| 7 | `Equal(f(1), f(2))`, two closures of one literal that capture 1 and 2 | Passes | Fails: two different functions |
| 8 | `Equal` of two `map[float64]int{NaN: 1}` | Panics, with `EquateNaNs` and without | Fails, and passes under `EquateNaNs` |
| 13 | `Invert` of `reading(-0)`, with the values +0, -0 and NaN registered | Returns the choice of +0 | Returns the choice of -0 |

Each defect has one cause:

- go-cmp calls a type's `Equal` method before it compares fields
  (`compare.go`, lines 59 to 62 and 265 to 268). It matches map keys
  with `==` and panics on a NaN key (lines 510 to 550). It compares two
  non-nil functions as unequal (lines 286 and 287), and `Options` adds a
  comparer by code pointer (`option.go`, lines 106 to 120).
- `contained` hashes the needle and matches it with `==`
  (`contains.go`, lines 176 to 181).
- `literalConverter` matches a Go value with `reflect.DeepEqual`, which
  compares floats with `==`, so -0 matches +0 first (`converter.go`,
  lines 724 to 728).

The same causes produce two defects that the review did not list:

- `prop.sameValue` decodes two typed literals as JSON and compares them
  with `reflect.DeepEqual`. A recorded -0 and a replayed +0 then compare
  alike, and the store reports no difference.
- `engine.canonicalKey` follows pointers, maps and slices without a
  guard. `UniqueList(SampledFrom(a, b))` of two values that contain
  themselves ends the test binary with `fatal error: stack overflow`.

The writer renders the diff of a failure with go-cmp, under go-cmp's
rules. The text of a failure can then contradict its verdict: it
shows no difference where an `Equal` method equates two values, and the
writer recovers from go-cmp's panic on a NaN key.

The comparisons also allocate for results that a passing call discards.
`Equal` builds a diff for every call. `Options` builds the exporter and
the function comparer on every call, and `contained` builds them again
for every element. By their allocation contracts, a passing `Equal` of
two ints allocates 24 times, `Contains` of a slice of three ints 76
times, and `Permutation` of two slices of three ints 120 times.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| Structural equality | `internal/equality`, new | `equal`'s rules over two values, the key lookup of `contains`, and the differences of two unequal values |
| Alignment | `internal/align`, new | The longest run of equal elements that two sequences share in order: the elements of two slices in a diff, and the lines of two texts |
| Relaxations | `internal/matcher/option.go` | `Option` returns `equality.Rules` with one flag set. Each assertion builds its rules once per call, by value |
| Verdicts | `internal/matcher` | `Equal`, `NotEqual`, `Contains`, `NotContains`, `Permutation`, `Pure` and the relation family call `equality.Equal` and `equality.HasKey` |
| Diff | `internal/matcher/writer.go` | Renders each difference of a failure of `equal` and of the golden comparisons, and a line diff of two texts of more than one line |
| Text of a reflect value | `internal/text` | `Sprintf` takes a `reflect.Value` as fmt does, so the writer renders a value of an unexported field |
| Property identity | `internal/prop/engine/value.go`, `inverse.go` | `SameValue`, exported, and a canonical key that terminates on every value |
| Walk path | `internal/cycle`, new | The pointers, maps and slices that enclose the position of a walk, for the canonical key and for `internal/text` |
| Literal inverse, store | `prop/converter.go`, `prop/store.go` | Call `engine.SameValue` and `literal.Canonical` |
| Corpus check | `conformance/corpus.go` | `checkRecord` compares a record's value by its canonical text, as `checkCall` compares a call record's |
| Suite check | `internal/matchertest/case.go` | `Verdict` compares a record's detail with a comparison of its own |

`internal/align` and `internal/cycle` import the standard library alone.
`internal/equality` imports the standard library and `internal/align`.
`internal/matcher` imports `internal/equality`, `internal/align` and
`internal/text`, and `internal/text` and `internal/prop/engine` import
`internal/cycle`.

```go
// Rules are the relaxations of one comparison. The zero value applies
// none, which is equal's default.
type Rules struct {
	// EquateEmpty makes a nil slice or map equal an empty one of its type.
	EquateEmpty bool
	// EquateNaNs makes a NaN equal a NaN of its type.
	EquateNaNs bool
}

// Equal reports whether x and y are equal under r.
func Equal(x, y reflect.Value, r Rules) bool

// HasKey reports whether the map m has a key that equals key under r.
func HasKey(m, key reflect.Value, r Rules) bool

// Diff returns the places where x and y differ under r, at most limit of
// them.
func Diff(x, y reflect.Value, r Rules, limit int) []Difference
```

### The rules of `equal`

A value of an interface type compares by the value inside it, and a nil
interface compares as the invalid value.

| Kind | Two values are equal when |
|---|---|
| Different types | Never. `int(1)` differs from `int64(1)` and from `1.0` |
| Bool, integer, string | `x == y` |
| Float, and each part of a complex number | `x == y`, so -0 equals +0 and NaN equals nothing. Under `EquateNaNs`, a NaN equals a NaN of its type |
| Channel, unsafe pointer | They are the same channel or the same address |
| Function | Both are nil, or they have the same code pointer |
| Pointer | Both are nil, or their targets are equal |
| Interface | Both are nil, or the values inside them are equal |
| Array, struct | Every element is equal, or every field, exported and unexported |
| Slice | Both are nil, or neither is nil and they have the same length and equal elements. Under `EquateEmpty`, nil equals empty |
| Map | Both are nil, or neither is nil, they have the same length, and a one-to-one matching pairs each entry with an entry whose key and value are equal. Under `EquateEmpty`, nil equals empty |

`Equal` meets these invariants:

- **It calls no method of a value.** It reads every field with
  `reflect`. A type's `Equal`, `String` or `Format` method cannot change
  the verdict or panic inside it.
- **A pointer equals itself only when its target does.** A pointer
  compared with itself compares its target with itself, so a pointer to
  a NaN differs from itself, as the NaN does. A slice and a map compared
  with themselves compare their elements in the same way.
- **A cycle stops.** A pair of pointers, maps or slices met a second
  time compares equal, as `reflect.DeepEqual` treats it. Two values are
  equal when no finite walk of both finds a difference. A pair of a slice
  and a shorter slice of the same array is another pair.
- **Depth costs heap memory.** The walk compares the elements of arrays
  and slices, the fields of structs and the targets of pointers from a
  stack of frames of its own. A frame leaves the stack when its last part
  begins, so a list in the last field of its nodes keeps one frame. Two
  lists of 100,000 links compare under a goroutine stack limit of 1 MiB.
  A map compares its entries in a nested walk. Maps nested in maps use
  the goroutine's stack.
- **It never panics.** It reads values through `reflect` methods that
  accept unexported fields, and it never calls `Interface`.

### Map entries

A lookup type is a key type whose `==` agrees with `equal`: a bool, an
integer, a string, a channel, an unsafe pointer, a float or a complex
number without `EquateNaNs`, and an array or a struct of those. When
every key of one map is of a lookup type, `Equal` looks each of them up
in the other map with `MapIndex`, and compares the two values. Maps of n
entries cost n lookups. An interface key counts by the value inside it,
so a `map[any]int` of scalar keys takes the same path.

For every other map, `Equal` matches the entries instead. Each entry of
one map takes the first unmatched entry of the other whose key and value
are equal. `equal` is an equivalence on the values that equal themselves,
so this greedy match is exact, as RFC-0003 argues for `Permutation`. An
entry whose key or value differs from itself, such as a NaN key without
`EquateNaNs`, matches nothing. Maps of n entries cost at most n² trials.

A trial that fails discards the pairs that it assumed equal. A later
comparison then starts from the pairs of the trials that succeeded.

`HasKey` looks the needle up with `MapIndex` when the needle is of a
lookup type that the map's key type accepts: the key type itself, or an
interface that the needle's type implements. Every key equal to the
needle then has the needle's type, and `==` agrees with `equal` there.
`HasKey` compares any other needle with each key. A needle that no key
can equal, such as an `int64` in a `map[int]bool`, fails without a scan.
A needle that Go cannot hash, such as a slice, is never hashed.

### The methods of a value

**Verdict:** `Equal` calls no method of a value.

**Reason:** the definition states structural equality. A method decides
by its author's rule, which can equate two values whose fields differ,
as finding 6 shows. On the first call and on calls at triangular
numbers, go-cmp also calls a method with its arguments in both orders,
and panics when the two results differ (`compare.go`, lines 347 to 364
and 641 to 651).

**Consequence:** a `time.Time` compares its wall clock, its monotonic
reading and its location, as Go's `==` compares a `time.Time`. A test
that compares two instants writes `assert.True(t, got.Equal(want), msg)`,
or strips the monotonic reading with `Round(0)` and fixes the zone with
`UTC()` before it calls `Equal`.

**What would reverse it:** a relaxation in the definition that defers to
a type's own equality.

### Functions

**Verdict:** a function compares by its code pointer, and the Go overlay
declares the limit.

**Reasons:**

- `reflect` returns the code pointer of a function alone. Its
  documentation states that "functions with equal code pointers may not
  have identical behaviors when called".
- A closure's identity is the address of its closure object, which is
  the function value itself (`abi-internal.md`, "Closures"). Only unsafe
  code reads that address, and for a function in an unexported field
  only through the internal layout of `reflect.Value`. The definition
  compares fields "as deeply as the language reaches without unsafe
  access".
- Comparing only nil functions as equal, as go-cmp and
  `reflect.DeepEqual` do, breaks the other half of the rule. A struct
  that contains a callback would differ from itself.

Under the limit, two closures of one function literal compare equal
whatever they capture, and so do two method values of one method on
different receivers. A function declared at the top level, a method
expression and a closure that captures nothing compare exactly.

**What would reverse it:** an API of `reflect` that returns the closure
object of a function value.

### Differences

`Diff` walks two unequal values under the rules of `Equal`, and returns
each place where they differ:

```go
// Difference is one place where two values differ.
type Difference struct {
	// Path are the steps from the roots of the two values to the place.
	Path []Step
	// X and Y are the values at the place. A side without the place, as
	// for an element that only the other side has, is the invalid value.
	X, Y reflect.Value
}

// Step is one step of a path: a field of a struct, an element of an
// array or a slice, or the value of a map's key.
type Step struct {
	// Field is the name of a field, and empty for another step.
	Field string
	// Index is the index of an element, and -1 for another step.
	Index int
	// Key is the key of a map entry, and the invalid value for another
	// step.
	Key reflect.Value
}
```

The walk descends into the parts of two values of one type, and compares
each part that it does not descend into as `Equal` compares it:

- **Values of different types, scalars, functions and channels** differ
  as a whole, and so do a nil pointer, slice or map and a non-nil one.
- **Pointers** descend into their targets, and a pair met again differs
  nowhere, as in `Equal`.
- **Structs and arrays** descend into each field and element.
- **Slices** align their elements with `internal/align`. An element that
  one side has alone is a difference with an invalid side, at its index
  in the side that has it. A run of elements that each side has alone
  between two aligned elements pairs up in order, and each pair descends
  at its index in `x`.
- **Maps** pair their entries by key. When `==` agrees with `equal` on
  every key of both maps, a map index pairs them. Otherwise each entry of
  `x` pairs with an equal entry of `y` first, and then with an entry of an
  equal key, so a pair of equal entries differs nowhere. The walk
  descends into the values of each pair. An entry that one side has
  alone is a difference with an invalid side. The differences of a map
  follow the order of its keys: by value for numbers, strings and bools,
  by address for pointers and channels, and part by part for arrays,
  structs and interfaces.

The walk descends at most 1,000 levels, and states the two values at
that depth whole. It stops at `limit` differences.

Outside the alignment of slices and the pairing of map entries without a
map index, the walk visits each part once. Two lists of 100,000 links
that differ in their last link take 28 ms, and two maps of 100,000
string keys take 80 ms. Each step of a path links to the step that
encloses it. The walk allocates one step for each part that it enters,
and copies a path out only for a place that differs.

### Alignment

```go
// Edits returns the edits that align a sequence of n elements with one of
// m elements: the longest run of pairs of equal elements that the two
// share in order, and between them the elements that each has alone.
func Edits(n, m int, equal func(i, j int) bool) []Edit
```

`Edits` first takes the elements that both sequences share at their
start and at their end. It aligns the rest by the longest common
subsequence, in a table of one cell for each pair of elements. A rest of
more than 1,048,576 pairs takes no common subsequence: its elements
pair by index in a diff, and its lines show as removed and added. Of two
alignments of one length, the one that lists the elements of the first
sequence first wins.

### The text of a difference

The writer renders the differences of a failure of `equal` and of the
golden comparisons, each on a line of its own after the contract:

```text
the order is stored: (-want +got)
	.Lines[2].Quantity: -3 +4
	.Tags[1]: -"gift"
	.Notes["courier"]: +"leave at the door"
```

- A path renders a field as `.Name`, an element as `[2]`, and a key as
  `["gift"]` or `[7]`. A difference at the roots renders without a path:
  `-1 +2`.
- A string renders quoted. Any other value renders as `text.Sprintf`
  writes it under `%+v`, so a value that contains itself renders with its
  cycle marked, and a value of an unexported field renders without its
  methods.
- Two values whose texts are equal, such as `1` in two interfaces of
  `int` and `int64`, render with their types: `-int(1) +int64(1)`.
- Two strings of one type of which one has more than one line render as
  a line diff with three lines of context, each line marked `-`, `+` or a
  space, and `…` for the lines left out. A golden file's failure renders
  that way.
- The writer renders at most 64 differences, and `…` for more.

The writer renders under the default rules, because a record contains
what the definition states and an option is no part of it. A relaxation
only widens what counts as equal, so the text can show a difference that
the verdict ignored, and never misses one that it counted. Two values
that `Diff` finds no difference in render as fields, as every other
record does.

### Property values

The property engine already implements the definition's identity of
generated values. `engine.sameValue` compares two values of one type by
their canonical keys, and two values of different types by the canonical
text of their typed literals. This RFC exports it as `SameValue` and
calls it from `prop`:

- The literal inverse matches a Go value with `engine.SameValue`.
- The store compares a recorded value with a replayed one by decoding
  both typed literals and comparing their `literal.Canonical` texts, as
  `SameValue` compares two literals. A value that does not decode
  differs.

The canonical key starts the encoding of each value with a tag: no
value, a value, or a back-reference. A pointer, a map or a slice met
inside itself is a back-reference, the tag and the number of steps back
to the container that it repeats. The key of every value is then finite,
and two values have the same key exactly when they have the same
structure and the same cycles. The keys live in memory alone, so the tag
changes no vector and no file.

`internal/cycle` declares the path of containers that a walk is inside.
The text package keeps such a path for its cycle marks, and the
canonical key needs the same one, so the path moves into one package:

```go
// Path is the pointers, maps and slices that enclose the position of a
// walk, outermost first.
type Path struct{ inside []container }

// Enter adds the pointer, map or slice v to the path, and reports true.
// When the path contains v already, it adds nothing, and returns the
// number of steps back to v and false.
func (p *Path) Enter(v reflect.Value) (back int, entered bool)

// Leave removes the container that the walk entered last.
func (p *Path) Leave()
```

### The checks of the tests

The checks that decide whether the core is right must not use the core's
comparison:

- `conformance` compares a value of a failure record with the case's
  literal by its canonical text, `literal.Canonical`, as it already
  compares a value of a call record. An int and a float of one rendering
  differ, and every NaN is one value.
- `internal/matchertest` compares a record's detail with a comparison of
  its own: an error matches a stated error under `errors.Is`, a NaN
  matches a NaN, a `*big.Int` matches a stated one of its value, and any
  other value matches as `reflect.DeepEqual` compares it, at any depth.
  A case's detail contains no value that contains itself.

### Failure handling

| Condition | Behaviour |
|---|---|
| A type whose `Equal` method panics or disagrees with its fields | The method does not run. The fields decide |
| A map key that differs from itself, such as NaN without `EquateNaNs` | It matches no key, so the map equals no map, itself included |
| A needle that Go cannot hash | It compares with each key, and no hash runs |
| A value that contains itself | The pair met again compares equal, and the walk stops |
| A list nested deeper than a goroutine's stack allows for a recursive walk | The walk's own stack keeps one frame |
| Two values that differ at more than 1,000 levels down | The diff states the two values at level 1,000 whole |
| Two long lists that differ near their ends | The diff visits each link once |
| Two slices whose rest after the shared start and end exceeds 1,048,576 pairs | Their elements pair by index |
| A failure whose values differ in more than 64 places | The text states the first 64 and `…` |
| A canonical key of a value that contains itself | A back-reference ends the key |

### Allocation contract

Measured on the working tree, in an ordinary build and under
`-cover -coverpkg=./...` alike:

- `equality.Equal` allocates nothing for two ints, two slices of ints and
  two structs of ints, 3 times for two lists of three links, 6 times for
  two maps of two string keys, and 11 times for two maps of two
  interface keys of scalars.
- `equality.HasKey` allocates once for a string in a map of string keys,
  the copy of the value that the map index reads.
- `equality.Diff` allocates 4 times for two structs that differ in one of
  two fields: a step for each field, the steps of the difference's path,
  and the difference.
- An assertion builds its `Rules` as a value: an `Option` takes the rules
  and returns them with its flag set, so no pointer to them escapes. No
  passing call builds a diff.
- An assertion that compares values builds the interface of each value
  that it compares. Go builds the interface of a pointer and of an
  integer below 256 without allocating, and allocates one for most other
  values.

| Passing call | Before | After |
|---|---|---|
| `Equal`, `NotEqual`, `Pure` of two ints below 256 | 24 | 0 |
| `Contains` of a slice of three ints | 76 | 1 |
| `Permutation` of two slices of three ints | 120 | 2 |
| `Idempotent`, `Commutative`, `Associative`, `RoundTrip` of ints below 256 | 24 | 0 |
| `Deterministic` with results of one int below 256 | 744 | 0 |
| `StableOrder` with sequences of three ints | 2,852 | 62 |
| `NoDuplicates` of three ints | 72 | 1 |
| `NotPure` with two readings of 256 or more | 26 | 2 |
| A run of 100 cases of `prop.IsPermutation` | 50,848 | 2,542 |
| A run of 100 cases of `prop.Deterministic` | 75,233 | 833 |

The `Test<Subject>Allocs` tests and the benchmark contracts state each
ceiling.

### Testing

- `internal/equality` is below the assertions of this module. Its tests
  use the package `testing` alone, as the tests of `internal/matcher`
  do. They cover every row of the rules table under
  both relaxations, each finding of the motivation, a cycle of one node
  against a cycle of two, two lists of 100,000 links under a goroutine
  stack limit of 1 MiB, a type whose `Equal` method panics, and a key
  that a failed trial assumed equal.
- The tests of `Diff`, `internal/align` and the writer cover each kind of
  difference, an element inserted into a slice, the order of a map's
  differences, a cycle, the limit, and the line diff of two texts. A
  chain through a pointer, an array, a slice and a map each reaches the
  depth of 1,000 levels at its 333rd link.
- The mutation stage of `ergon check` gates `internal/equality` at a
  100% score and 100% mutator coverage, as it gates `internal/matcher`.
- The shared suites of `internal/matchertest` gain the cases of findings
  5 to 8, so `internal/matcher`, `assert` and `expect` run them alike.
- `prop` gains a case for each defect of its own: -0 runs back to its
  own choice, and a recorded -0 differs from a replayed +0. The tests of
  the property engine's canonical key state that a unique list of values
  that contain themselves terminates.

### Definition changes

The definition changes first, and the module vendors it. RFC-0017 of the
definition moves it to 3.0.0:

- `overlays/go.json` gains a limit on `equal`. Its `what` states the
  limit of the section on functions and names every assertion that
  compares as `equal` compares. Its `why` states that Go exposes no
  identity of a closure without unsafe access.
- The corpus gains 12 cases that pin map keys: `contains`,
  `not-contains`, `equal` and `not-equal` of a NaN key, a negative zero
  key and a null key, by default and under `equate-nans`. A language
  whose map lookup equates NaN keys, as JavaScript's SameValueZero does,
  fails the cases without its relaxation.

### Standards and documentation

- The "Comparison rules" of RFC-0001 describe go-cmp's options. This
  RFC replaces them.
- `docs/standards/code.md` states that the module depends on the
  standard library alone, and that a dependency needs an RFC.
- `docs/standards/errors.md` drops `cmp.Diff` from the places where a
  `recover` exists.
- `docs/standards/testing.md` adds `internal/equality` to the packages
  that the mutation stage gates and whose tests use the package
  `testing` alone. It states that the verdicts of `conformance` and of
  `internal/matchertest` compare with code of their own, and never with
  `internal/equality`.
- The "Equality" section of the package documentation of `assert` and
  `expect`, and the "Equality" table of the README, state that no method
  runs, that a map key compares as a value compares, and the limit on
  closures. The README states no dependency.

## Alternatives considered

### A. go-cmp with more options

go-cmp's rules are options: `cmpopts.EquateNaNs`, comparers and
filters.

**Why not:**

- No option turns off the `Equal` method of every type. A comparer
  needs a concrete type, so each type with a method would need one.
- The panic on a NaN key is part of go-cmp's documented contract. Its
  help text asks the caller for a comparer of the whole map.
- `Contains` on a map would still need a rule of its own, outside
  go-cmp.

### B. go-cmp for the diff alone

`internal/equality` decides every verdict, and the writer renders the
diff with go-cmp.

**Why not:** the text of a failure would follow go-cmp's rules and not
the verdict's. It would show no difference where an `Equal` method
equates two values, show two structurally equal pointer keys as an entry
removed and one added, and need a `recover` for the panic on a NaN key.
The module would also keep a dependency for its text alone.

### C. `reflect.DeepEqual`

**Why not:**

- It compares two non-nil functions as unequal, so a struct with a
  callback differs from itself.
- It treats a pointer, a slice or a map as equal to itself whatever it
  contains, so a slice that contains a NaN equals itself.
- It matches map keys with `==`, and has no relaxation.

### D. Closure identity through unsafe access

Read the closure object's address from the function value.

**Why not:** `reflect.Value` exposes no address of a function in an
unexported field. Code would read it through the internal layout of
`reflect.Value`, which changes without notice. The layout of a function
value is the gc compiler's internal ABI. The definition also compares
fields without unsafe access.

### E. Map keys matched with `==`

Match the keys of two maps with `==`, as Go's own lookup, go-cmp and
`reflect.DeepEqual` match them.

**Why not:** the definition states that a key compares as `equal`
compares. Finding 5 breaks that rule. Under `EquateNaNs`, `==` also
misses a NaN key that the relaxation makes equal.

**What would reverse it:** the definition pinning the lookup of each
language's own maps.

### F. Cycles equal only at the same step

go-cmp compares two cyclic values as equal only when both walks meet
the repeated pair at the same step. A cycle of one node then differs
from a cycle of two nodes with the same contents.

**Why not:** the definition compares values, and those two cycles
unfold to the same value. `reflect.DeepEqual` treats a repeated pair as
equal. That rule needs no state per path. Outside map matching, it
compares each pair of containers once.

### G. One package for both equalities

`internal/equality` gains a rule that compares floats by their bits. The
property engine calls it.

**Why not:** a unique list needs a key that it can hash. The canonical
key is that key. A key and a comparison would then implement one
identity twice, and the two would drift apart.

### H. A line diff of the two values' texts

Render want and got as text, one part on a line, and diff the lines, as
many assertion libraries do.

**Why not:** two values can differ where their texts agree: an `int` and
an `int64` in two interfaces, or two NaNs. A diff of texts then shows no
difference for a failing verdict. The walk of `Diff` follows the rules
of the verdict, and adds the types where two texts agree.

## Drawbacks

- A test that compared two `time.Time` values through their `Equal`
  method fails on a monotonic reading or a different location.
- Two maps of n entries whose keys need matching cost up to n² trials.
- The diff states each difference at its path, where go-cmp writes the
  two values as Go literals with the differences marked. A reader sees
  less of the values around a difference.
- Two slices whose rest exceeds 1,048,576 pairs of elements pair their
  elements by index, so an element inserted early in a long slice shows
  as a difference at every later index, up to the limit.
- A failure walks its values twice: once to decide the verdict, and
  once to find its differences.
- The alignment of two slices compares their elements with `Equal`, so
  slices nested in slices compare the elements of each level again. A
  chain of n nested slices costs up to 1,000·n comparisons.
- Two closures of one literal compare equal. The overlay declares it,
  and no Go test of the definition can tell them apart.
- `internal/equality` adds a package to the mutation stage of CI.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| The equality of `equal` | Definition 3.0.0, RFC-0001, "Equality", and `spec/assertions.yaml` |
| The pinned map keys | Definition 3.0.0, RFC-0017 |
| The identity of generated values | Definition 3.0.0, RFC-0010, the unique collections |
| The typed literals of maps and floats | Definition 3.0.0, `spec/encoding.md` |
| The overlay's limits | Definition 3.0.0, `spec/overlays.md` |
| go-cmp's rules and panics | go-cmp v0.7.0, `cmp/compare.go`: `Equal` and its rules, `tryMethod`, `callTTBFunc` and `compareMap` |
| `reflect.DeepEqual` | Go 1.27.1, `src/reflect/deepequal.go` |
| The code pointer of a function | `reflect.Value.Pointer`, Go 1.27.1 |
| The closure object | Go 1.27.1, `src/cmd/compile/abi-internal.md`, "Closures" |
| The greedy match of an equivalence | RFC-0003, "Comparison" |
| The rules this RFC replaces | RFC-0001, "Comparison rules" |
