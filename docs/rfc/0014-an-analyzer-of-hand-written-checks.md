---
rfc: 0014
title: An analyzer of hand-written checks
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-06
updated: 2026-10-07
discussion: https://github.com/dokimasia/assert-go/issues/19
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0014: An analyzer of hand-written checks

## Summary

A test that states a condition through `True` or `False`, or through an
`if` statement and `t.Fatalf`, hides the assertion that states the
condition. Its failure reports the message alone, or the text that the test
formats. The assertion reports the record of the values that differ. A
review of two modules' tests replaced 54 calls of `True` and `False`, 26
sequences of an encode, a parse and an `Equal`, and 17 checks of an ended
context with the assertions that state them, each found by hand.

This proposal adds a module of its own, `go.dokimi.dev/assert/lint`, in the
directory `lint/` of this repository:

- **The package `lint` exports `Analyzer`,** a `go/analysis` analyzer
  named `assertlint`. It has a rule for each assertion of the definition
  whose check a test writes by hand in a form that the type checker
  recognises: 61 rules over 100 of the 110 assertions. 21 of the rules
  suggest a fix, wherever the rewrite keeps the check's meaning in every
  case.
- **The command `cmd/assertlint`** runs the analyzer through
  `singlechecker`. `assertlint ./...` reports, and `assertlint -fix ./...`
  applies the fixes. `go vet -vettool` and `go fix -fixtool` run it as well.
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
`InRange(t, n, 1, 1<<63, "the count is positive")` fails with the count.
Most of the conditions are syntactic patterns, and a type checker tells the
safe rewrites from the others.

### Checks that tests write without the library

A test that does not import the library writes each check as an `if`
statement: `if err != nil { t.Fatal(err) }` and `if got != want {
t.Errorf(...) }`. Its allocation, golden and benchmark checks take the same
form, such as `if testing.AllocsPerRun(100, f) != 0 { t.Fatal(...) }`. The
analyzer reads such a statement as `False` of its condition, so every rule
over a condition reports both forms.

### A module of its own

`go/analysis`, its test harness and `singlechecker` are packages of
`golang.org/x/tools`. The standards admit a dependency of the module
`go.dokimi.dev/assert` only through an RFC. A module of its own keeps the
requirement out of the module graph of every consumer of the assertions. A
test that imports `assert` loads no package of `x/tools`. A run of the
analyzer loads them.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| The analyzer | `lint/analyzer.go` | `Analyzer`, the walk of a package, the order of the rules, and the report of a diagnostic |
| The checks | `lint/check.go` | The checks that a rule reads: a call of an assertion, an `if` check, its condition, and an equality |
| The origins | `lint/origin.go` | The calls that produce the value of a variable |
| The rules | `lint/truth.go`, `equal.go`, `nilness.go`, `length.go`, `errors.go`, `contains.go`, `text.go`, `numeric.go`, `order.go`, `panics.go`, `behaviour.go`, `relation.go`, `waiting.go`, `measurement.go`, `rejects.go`, `golden.go`, `files.go`, `prop.go`, `chain.go` | The rules, one file for each family of the assertions that they suggest |
| The edits | `lint/edit.go` | The text of a call rewritten from the source of its arguments |
| The annotations | `lint/skip.go` | The comments that leave out the reports of a line, and the report of an annotation that leaves out none |
| The command | `lint/cmd/assertlint/main.go` | `singlechecker.Main(lint.Analyzer)` |
| The test module | `lint/testdata/` | A module that requires `go.dokimi.dev/assert` through a `replace` of the repository's root, with one package for each assertion and its golden files |

### The public surface

```go
package lint

// Analyzer reports a check that a test writes by hand and that an
// assertion of go.dokimi.dev/assert states, and suggests the rewrite where
// the rewrite keeps the check's meaning in every case.
var Analyzer = &analysis.Analyzer{
	Name:     "assertlint",
	Doc:      "report hand-written checks that an assertion states",
	URL:      "https://pkg.go.dev/go.dokimi.dev/assert/lint",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}
```

Each diagnostic starts with the name of its rule and names the assertion,
such as `compare: state the check with Equal`. A rule that matches calls
outside the check names them too, because the check's line does not show
them, as in `round-trip: state the check with RoundTrip of json.Marshal and
json.Unmarshal` and `pure: state the check with Pure of s.Snapshot(), around
s.Get("key")`. A diagnostic quotes each call on one line and cuts a call
longer than 60 characters to its function, as `sort.Slice(…)`.

### The checks that the analyzer reads

A rule reads one of three kinds of check:

- **A call of an assertion.** The analyzer recognises a package-level
  function of `go.dokimi.dev/assert` or `go.dokimi.dev/assert/expect` whose
  first parameter is `assert.TB`, by the function that the type checker
  resolves.
- **An `if` check.** This is an `if` statement without `else` whose body is
  one call of `Error`, `Errorf`, `Fatal`, `Fatalf`, `Fail` or `FailNow` on
  a test, a value whose method set has `Helper` too. It states that its
  condition is false, as `False` of the condition states it.
- **A statement that a rule names.** Examples are a loop that sleeps until
  it ends on a condition, a `select` with a case of `time.After`, and a call
  of `testing/quick`.

A condition loses its parentheses and its negations, and each negation
inverts what the check states. `False(t, !(x == nil))` and `if x != nil {
t.Fatal() }` both state `x == nil`.

An equality is a check that two values are equal or differ. `Equal` and
`NotEqual` without options are equalities. So is a condition of `==`,
`!=`, `reflect.DeepEqual`, `bytes.Equal`, `slices.Equal` or `maps.Equal`.
A rule over the operands of an equality, such as `golden-match`, reads
every form alike. The rules from `commutative` to `permutation`, except
`not-pure`, read an equality that states its values equal. The rule
`not-pure` reads one that states them different.

The origin of a variable is the call that assigns it, as `os.ReadFile` in
`want, err := os.ReadFile(path)`, or the call that receives its address,
as `json.Unmarshal` in `json.Unmarshal(data, &got)`.

An inert statement is a call of an assertion, whatever its arguments call,
or a declaration whose variables take no values. The rules `round-trip` and
`after-close` count no inert statement as a step between two calls.

The rules over two readings of one call, from `deterministic` to
`idempotent`, count an assertion as a step where its values make a call
that can write a variable that the reading reads:

- A method of the variable, as `s.Put(1)` beside `s.Total()`.
- A call that receives the variable's address, as `load(&cfg)` beside
  `sum(cfg)`.
- A call that receives a value of the variable that shares memory, as a
  slice, a map or a pointer does. A function literal that passes `l` to
  `WriteManifest` beside `l.Writes()` makes such a call.

A call of an assertion or of a method of a test writes nothing. A test, which
every assertion and helper takes, counts as no variable of a reading.

A rule that relates a value to the call that made it follows the value only
through earlier statements of the check's block. A value that a loop
assigns has no origin there, as in a benchmark's loop. A function literal
that assigns the value ends the search, as the function of `MaxAllocs`
does, because the literal gives the value when it runs. These rules are
`round-trip`, the context rules, `path-absent`, `after-close`,
`has-content` and `golden-match`. Of them, the context rules follow the
literal, as the table of rules states. Other rules, such as `links-to`,
follow a value through at most four conversions and origins anywhere in the
package.

### The rules

| Rule | Reports | Suggests | Fix |
|---|---|---|---|
| `panics`, `not-panics` | `Nil`, `NotNil`, or a comparison with nil, of a value whose origin is `recover()` | `Panics`, `NotPanics` | No |
| `nil-context-safe` | `NotPanics` of a function literal that passes nil for a parameter whose declared type is `context.Context` | `NilContextSafe` | No |
| `in-range` | `True` of `lo <= x && x <= hi`, or `False` of `x < lo \|\| x > hi`, over one number x, in any order of each comparison's operands | `InRange` | Where x calls no function and both bounds are exact as a float64 |
| `conjunction` | `True` of `a && b` and `False` of `a \|\| b`, as a call or as an `if` check outside a loop that waits for a condition, as `eventually` reads it | The assertion that each operand takes alone, as the rules over a check name it for the operand | For a statement of assert, a `True` or a `False` of each operand |
| `honours-cancellation` | A match of an error with `context.Canceled`, where the call that returns it receives a context whose cancel function an earlier statement called, directly or through a function of the package that returns such a context. A function literal of an earlier statement that assigns the error returns it where each of its assignments of the error is such a call with a context that ended before the literal, and no statement from the literal up to the check assigns the context's variable | `HonoursCancellation` | No |
| `honours-deadline` | A match of an error with `context.DeadlineExceeded`, where the call that returns it receives a context whose deadline had passed when it was made, directly or through a function literal as for `honours-cancellation` | `HonoursDeadline` | No |
| `path-absent` | `True` of `os.IsNotExist(err)`, and a match of err with `fs.ErrNotExist` or `os.ErrNotExist`, where err comes from `os.Stat` | `files.Absent` | No |
| `after-close` | A match of an error with a sentinel, where the error comes from a method of a value whose `Close` an earlier statement called, with only inert statements between the two | `FailsAfterClose` | No |
| `errors-is` | `True` or `False` of `errors.Is(err, target)` | `ErrorIs`, `ErrorIsNot` | Yes |
| `errors-as` | `True` of `errors.As(err, &target)`, or of a value whose origin is that call | `ErrorAs` | For a statement of assert whose target's type the file can write, and whose target another statement reads |
| `sentinel` | `err == target` and `Equal(err, target)`, where exactly one operand is a package-level variable of an error type | `ErrorIs`, `ErrorIsNot` | No |
| `is-dir`, `is-file` | `True` of `IsDir()` of an `fs.FileInfo` or an `fs.FileMode`, and of `IsRegular()` of an `fs.FileMode` | `files.IsDir`, `files.IsFile` | No |
| `has-mode` | An equality of `Perm()` of an `fs.FileMode` | `files.HasMode` | No |
| `links-to` | An equality of a value whose origin is `os.Readlink` | `files.LinksTo` | No |
| `has-content` | An equality of the text that `os.ReadFile` reads from a file outside `testdata` | `files.HasContent` | No |
| `golden-match` | An equality of the text that `os.ReadFile` reads from a file under `testdata` | `golden.Match`, `golden.MatchAt` | No |
| `max-allocs` | A check whose value under test is a value of `testing.AllocsPerRun` | `MaxAllocs`, `MaxAllocsWithSetup` | No |
| `goroutine-leaks` | A check whose value under test is a value of `runtime.NumGoroutine` | `NoGoroutineLeaks` | No |
| `bench-max-allocs`, `bench-max-bytes`, `bench-max-mean` | A check whose value under test is `AllocsPerOp`, `AllocedBytesPerOp` or `NsPerOp` of a `testing.BenchmarkResult`, or `Elapsed` of a `*testing.B` | `bench.Contract.MaxAllocs`, `MaxBytes`, `MaxMean` | No |
| `commutative` | An equality of `f(a, b)` and `f(b, a)` | `Commutative` | For `Equal`, where f is no builtin and has no type parameters, and neither f nor an operand calls a function |
| `associative` | An equality of `f(f(a, b), c)` and `f(a, f(b, c))` | `Associative` | For `Equal` with the left grouping first, under the same conditions |
| `round-trip` | An equality of x and `g(f(x))`, where x is no constant, f is no function of a test file that takes no test, every other statement from the call of f up to the check is inert, and only g reads the result of f | `RoundTrip` | No |
| `deterministic` | An equality of the results of two consecutive calls of one function with one input, other than two errors, with no step between them, where the second call may be in the check | `Deterministic` | No |
| `stable-order` | The same for a function without parameters that returns a slice other than bytes | `StableOrder` | No |
| `pure`, `not-pure` | An equality of the results of two calls of one function, other than two errors, with steps between them. Where a step passes the first result to a call that can write it, the check observes a copy of that result | `Pure`, `NotPure` | No |
| `idempotent` | The same, where one statement repeats before each call | `Idempotent` | No |
| `permutation` | An equality of two slices that earlier statements sort | `Permutation` | No |
| `rejects` | A check of `Failed()` of a test other than the check's own, such as an `assert.Recorder` | `Rejects` | No |
| `equal-func` | `True` or `False` of `bytes.Equal`, or of `slices.Equal` or `maps.Equal` over booleans, numbers or strings | `Equal`, `NotEqual` with `EquateEmpty` | Where both operands have one type |
| `deep-equal` | `True` or `False` of `reflect.DeepEqual(a, b)` | `Equal`, `NotEqual` | Where both operands have one type that contains no float, complex number, function, interface or `unsafe.Pointer`, and no pointer in a map key |
| `nil` | `x == nil` or `x != nil` for x a pointer, slice, map, channel, function or `unsafe.Pointer` | `Nil`, `NotNil` | Yes |
| `error-nil` | The same for x of an interface type that implements `error` | `NoError`, `HasError` | Yes |
| `equal-nil` | `Equal` or `NotEqual` of a value and nil, without options | `Nil`, `NotNil`, `NoError`, `HasError` | Yes, for the types of `nil` and `error-nil` |
| `length` | `Equal(len(x), n)`, `NotEqual(len(x), 0)`, `Length(x, 0)`, and a comparison of `len(x)` with a count, where `len(x)` is the value under test | `Length`, `Empty`, `NotEmpty` | Yes, but `Length` of a string |
| `contains` | `strings.Contains`, `bytes.Contains`, `slices.Contains`, and the `ok` of a map lookup | `Contains`, `NotContains` | Yes, but the map lookup, and `slices.Contains` over bytes or over values that are no boolean, number or string |
| `membership` | `True` of `x == a \|\| x == b`, or `False` of `x != a && x != b`, over one variable x | `Contains` | Where x is a boolean, a number other than a byte, or a string, whose type the file can write, and no member calls a function |
| `contains-in-order` | `strings.Index(s, a) < strings.Index(s, b)` | `ContainsInOrder` | No |
| `has-prefix`, `has-suffix` | `strings.HasPrefix`, `bytes.HasPrefix` and their suffix forms | `HasPrefix`, `HasSuffix` | Yes |
| `matches` | `regexp.MatchString`, and `MatchString` or `Match` of a `*regexp.Regexp` | `Matches` | No |
| `close-to` | `math.Abs(a-b) <= tol` and `math.Abs(a-b) < tol` | `CloseTo` | For `<=` |
| `pairwise` | `slices.IsSorted`, `slices.IsSortedFunc`, and the functions of `sort` that report whether values are sorted | `Pairwise` | No |
| `order` | `<`, `<=`, `>` or `>=` between a number and a constant | `InRange` | For an integer and a constant of a magnitude up to 2^53 |
| `compare` | `a == b` or `a != b` between booleans, numbers or strings | `Equal`, `NotEqual` | Yes |
| `condition` | An `if` check that no other rule reports, outside a loop that waits for a condition | `True`, `False`, and for a guard on `a && b` the assertion of `!b` under an `if` of `a` | No |
| `total` | A range over a slice whose body is one `NoError` of `f(element)`, where f does not read the element | `Total` | For assert, where f calls no function and the message does not read the element |
| `poisoned` | A counted loop whose body is one `HasError` of a call | `Poisoned` | No |
| `no-duplicates` | A range whose body checks `seen[x]` and assigns `seen[x]` | `NoDuplicates` | No |
| `monotonic` | A loop that checks the order of a value and of the value that it keeps from the step before | `Monotonic` | No |
| `eventually` | A loop that waits for a condition: it calls `time.Sleep`, a return or a break of its body or the condition of a `for` statement that compares no counter can end it, and a test is in scope | `Eventually`, `EventuallyTrue` | No |
| `completes-within` | A `select` whose case of `time.After` fails the test | `CompletesWithin` | No |
| `update-flag` | A definition of the flag `-update` in a test file | `golden.MatchAt` with `golden.ShouldUpdate` | No |
| `for-all` | A loop that calls a function of `math/rand` or `math/rand/v2` and checks, and a call of `testing/quick` | `prop.ForAll` | No |
| `property-form` | A `ForAll` whose body assigns one value of `Draw` and calls one assertion that has a property form, where the form takes each argument that reads the drawn value: as a function of the input in place of a value or of a function with fewer parameters, or as an argument that the form generates | The form, such as `prop.Equal` | No |
| `machine` | A `ForAll` whose body loops over actions and switches on a value of `Draw` | `stateful.Steps` | No |
| `chain` | Two or more consecutive statements that each call an assertion of one surface on one variable, where the surface's chain has a method of the assertion's name | `That` | Where no other fix edits one of the calls, and no comment is between them |

The rules over one call or `if` check run in the order of the table, and
the first that reports the check ends the search. A rule over a statement,
from `total` to `chain`, reads its statement apart from them.

### Every assertion

Each assertion of the definition has a package of the test module. The
package calls the assertion as intended, and no rule may report the call.
Where a rule recognises a hand-written form of the assertion's check, the
package writes each form, and the rule must report it.

| Assertion | Rules |
|---|---|
| `equal`, `not-equal` | `compare`, `deep-equal`, `equal-func` |
| `true`, `false` | `condition` |
| `nil`, `not-nil` | `nil`, `equal-nil` |
| `err-absent`, `err-present` | `error-nil`, `equal-nil` |
| `length`, `empty`, `not-empty` | `length` |
| `contains`, `not-contains` | `contains`, `membership` |
| `contains-in-order` | `contains-in-order` |
| `permutation` | `permutation` |
| `has-prefix`, `has-suffix` | `has-prefix`, `has-suffix` |
| `matches` | `matches` |
| `close-to` | `close-to` |
| `in-range` | `in-range`, `order` |
| `pairwise` | `pairwise` |
| `err-is`, `err-is-not` | `errors-is`, `sentinel` |
| `err-as` | `errors-as` |
| `throws`, `not-throws` | `panics`, `not-panics` |
| `honours-cancellation` | `honours-cancellation` |
| `honours-deadline` | `honours-deadline` |
| `completes-within` | `completes-within` |
| `pure`, `not-pure` | `pure`, `not-pure` |
| `nil-context-safe` | `nil-context-safe` |
| `idempotent` | `idempotent` |
| `deterministic` | `deterministic` |
| `commutative` | `commutative` |
| `associative` | `associative` |
| `round-trip` | `round-trip` |
| `stable-order` | `stable-order` |
| `no-duplicates` | `no-duplicates` |
| `monotonic` | `monotonic` |
| `total` | `total` |
| `after-close` | `after-close` |
| `poisoned` | `poisoned` |
| `eventually`, `eventually-true` | `eventually` |
| `no-task-leaks` | `goroutine-leaks` |
| `max-allocs`, `max-allocs-with-setup` | `max-allocs` |
| `golden-match`, `golden-match-at` | `golden-match`, `update-flag` |
| `has-content` | `has-content` |
| `has-mode` | `has-mode` |
| `is-dir`, `is-file` | `is-dir`, `is-file` |
| `links-to` | `links-to` |
| `path-absent` | `path-absent` |
| `bench-max-allocs`, `bench-max-bytes`, `bench-max-mean` | `bench-max-allocs`, `bench-max-bytes`, `bench-max-mean` |
| `rejects` | `rejects` |
| `prop-for-all` | `for-all` |
| Each of the 39 property forms, from `prop-accumulates` to `prop-true` | `property-form` |
| `accumulates` | None |
| `bench-max-latency` | None |
| `golden-match-json-field`, `golden-match-tree` | None |
| `tree-equal`, `tree-contains`, `tree-unchanged` | None |
| `linearizable`, `serializable`, `snapshot-isolation` | None |

An assertion without a rule has no hand-written form that a rule can tell
apart from other code:

- **`accumulates`** subtracts readings across statements, and no
  syntactic pattern matches that code.
- **`bench-max-latency`** needs a percentile, which a test computes over
  its own samples with code of its own.
- **`golden-match-json-field`, `golden-match-tree`, `tree-equal`,
  `tree-contains` and `tree-unchanged`** decode a document or walk a
  directory, and no syntactic pattern matches that code.
- **`linearizable`, `serializable` and `snapshot-isolation`** have no
  hand-written form. A test checks a history with a checker, not with a
  condition.

The machines of `stateful` have a package too, whose `ForAll` loops over
actions that `Draw` selects, and the rule `machine` reports it.

### Fixes

A fix applies only where the rewrite keeps the check's meaning in every
case:

- **An `if` check gets no fix.** Its failure text states what went wrong,
  and an assertion's message states the contract.
- **`nil` and `equal-nil` leave out other interfaces.** `Nil` counts a typed
  nil in an interface as nil. `==` and `Equal` count it as present.
  `NoError` reports a typed nil as an error, so an `error` keeps its
  meaning.
- **`compare` leaves out pointers, structs and arrays.** `Equal` compares
  their contents. `==` compares a pointer by its address.
- **`deep-equal` requires a type without floats, complex numbers,
  functions, interfaces and unsafe pointers.** `reflect.DeepEqual` counts a
  value equal to itself through one pointer. A NaN behind one pointer is
  then equal to itself, while `Equal` compares the NaNs. `reflect.DeepEqual`
  compares two functions as unequal. It finds a map key through `==`.
- **`equal-func` adds `EquateEmpty`,** because `bytes.Equal`, `slices.Equal`
  and `maps.Equal` report a nil value equal to an empty one.
- **`contains` leaves out `slices.Contains` over bytes,** because
  `Contains` reads a `[]byte` as text.
- **`length` writes `Length` for no string.** `len` counts a string's bytes,
  and `Length` counts its Unicode scalar values.
- **`conjunction` and `total` fix a statement of assert alone.** A failed
  assertion of assert ends the test, so a later operand runs only where the
  earlier ones are true, as `&&` runs them. A failed assertion of expect
  continues, so an operand that an earlier one guards, such as `p.N` after
  `p != nil`, would run on a nil p.
- **`total` fixes a loop whose function calls nothing.** `Total` evaluates
  its function once, and the loop evaluated `f.store().Add` for each
  element. The fix also requires a message that does not read the element,
  which is out of scope after the loop. The record of `Total` states the
  index and the error of the element that fails.
- **`errors-as` fixes a statement of assert alone.** On a failure,
  `ErrorAs` returns the zero value. The original leaves the target
  unchanged. The fix also requires a statement that reads the target. A
  local variable that the fix assigns and no statement reads does not
  compile.
- **`order` and `in-range` keep each bound exact.** `InRange` reads its
  value as a float64. A float64 represents every integer of a magnitude up
  to 2^53 exactly. A strict order of integers moves its constant by one. A
  strict order of floats gets no fix.
- **`order` and `in-range` keep the source of a bound.** A literal bound
  becomes the number of the closed bound. Any other bound keeps its source
  text, with `+1` or `-1` after it, so the check changes with a named
  constant: `took < slowStart` becomes
  `InRange(t, took, -1<<63, float64(slowStart-1), msg)`. A typed bound gets
  a conversion to float64, which the parameter of `InRange` takes.
- **`close-to` fixes `<=` alone,** because `CloseTo` passes a difference
  equal to the tolerance.
- **`commutative`, `associative` and `membership` change how often an
  operand runs.** `Commutative` and `Associative` evaluate each operand
  once, and the slice of `Contains` evaluates every member, where `||`
  stops at the first equal one. An operand must not call a function or
  receive from a channel.
- **`chain` replaces its statements,** so it applies only where no other fix
  edits them.
- **`sentinel` gets no fix,** because `==` compares the errors themselves,
  and `ErrorIs` also matches a sentinel that another error wraps.
- **`contains-in-order`, `matches`, `max-allocs`, `goroutine-leaks` and the
  bench rules change what the check measures.** A missing needle has the
  index -1, `Matches` reads the portable subset of regular expressions,
  `MaxAllocs` rounds its average to the nearest whole number,
  `NoGoroutineLeaks` follows the goroutines that its scope starts, and a
  contract measures its own iterations.
- **The other rules suggest no fix,** because the rewrite replaces
  statements that the test arranges in its own way.

A fix of a call that names its assertion through a dot import or with type
arguments is left out, because the fix renames the assertion through the
name of its package.

### Annotations

A reviewer can keep a check that a rule reports. An example is a loop over
a fixed seed that replays one workload, which `prop.ForAll` would run 100
times. A line comment leaves out the reports of the rules that it lists on
one line:

```go
//dokimi:lint-skip for-all: a fixed workload that the machine above checks
for range rounds {
```

- On a line of its own, the annotation covers the next line. After code,
  it covers its own line. A report belongs to the line where its node
  starts.
- The rules are a comma-separated list of rule names. The analyzer has no
  keyword for every rule, so an annotation lists each rule that it excuses.
- The reason after the colon is required. The run reports an annotation
  without rules or without a reason under the name `lint-skip`, and the
  annotation leaves out nothing.
- The run reports each listed rule that leaves out no report on the covered
  line. An annotation then fails the run once the check that it excuses is
  gone, and a misspelt rule fails it at once.
- A report that an annotation leaves out suggests no fix.

### The command and its coverage

The command is one statement. Its test builds the command with `-cover`
into a directory of the test. It runs the command over a copy of a package
of the test module, with `GOCOVERDIR` set to the directory of
`-test.gocoverdir`. `go test -cover` then merges the command's counters
with the test's. The command's statement counts as covered without a test
in the package `main`.

The test requires the exit status and the output of each mode:

| Mode | Exit status | Output |
|---|---|---|
| `assertlint ./...` | 3 | Each diagnostic |
| `assertlint -fix ./...` | 0 | Nothing, and the fixed file |
| `go vet -vettool=<command> ./...` | 1 | Each diagnostic |
| `go fix -fixtool=<command> ./...` | 0 | Nothing, and the fixed file |

Under `-fix`, `singlechecker` applies the fixes and prints no diagnostic. A
second run without `-fix` reports the checks that have no fix.

### The repository

- The workspace file `go.work` lists both modules. `ergon` reads its
  module set from the workspace, so `ergon check` lints, tests and covers
  `lint` as it does the root.
- The coverage stage of `ergon` runs `go tool cover` from the root over
  the profiles of both modules. The workspace resolves the packages of
  `lint` there, and the root module alone does not.
- The test module under `lint/testdata` stays outside the workspace.
  `analysistest` loads it with `GOWORK=off`, and the command's test runs
  the command and the go command with `GOWORK=off` as well.
- The mutation stage mutates only `internal/matcher`, `internal/equality`
  and `conformance`.
- The page of `go.dokimi.dev/assert/lint` on the host `go.dokimi.dev`
  serves the same `go-import` tag as the page of `go.dokimi.dev/assert`.
  The host serves no page for that path today, and a consumer outside this
  repository resolves the module through it.
- A release of the analyzer is a tag `lint/vX.Y.Z`.
- The module does not require `go.dokimi.dev/assert`. Its tests check the
  diagnostics and the fixes through `analysistest`, and the command through
  its exit status, its output and the files that it writes. A requirement
  would need a released version of the assertions, because `go install` of
  a module refuses a `replace` directive.

### Testing

- One package of the test module per assertion, under
  `lint/testdata/<assertion id>/`, whose `// want` comments state each
  diagnostic and whose golden files state each fix. The package `lint-skip`
  states the annotations.
- The spec of the file whose rules report a package runs it.
  `analysistest.RunWithSuggestedFixes` runs the packages of one spec in one
  run, and the spec reports the errors of each package in a subtest of its
  case. `analysistest` parses and type-checks every dependency from source
  in each run. One run for each spec takes the module's tests from 75 s to
  16 s under `-race`, measured on four cores.
- A golden file must receive a fix, because `RunWithSuggestedFixes` reads
  the golden file of a file with a fix alone.
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
that tell an interface from a pointer, follow an origin, or compare
statements of a block need the type checker's results, which an analyzer
reads directly.

### C. A plugin of golangci-lint

The module would register the analyzer with
`github.com/golangci/plugin-module-register`, and a custom build of
golangci-lint would run it.

**Why not:** the module would require a second dependency for one runner.
`go vet -vettool`, `go fix -fixtool` and the command run the analyzer
without it, and a custom build can import `lint.Analyzer` itself.

### D. Fixes for every rule

`if` checks would get a message, and `matches` would rewrite to `Matches`.

**Why not:** a rewrite that changes what a check accepts turns a lint run
into a broken test. An `if` check states no contract to use as the message.
The diagnostic names the assertion, and the author writes the call.

### E. Rules for other libraries

`cmp.Diff` of go-cmp, the assertions of testify, and the checkers of
porcupine and rapid would each get rules.

**Why not:** each library needs a rule set of its own, and the test module
would need a copy of each. `docs/migrating.md` covers the move from
testify.

### F. A list of packages to leave out

A setting of the command would list the packages whose reports a run
leaves out.

**Why not:** a package list leaves out every check of the package with the
one that a reviewer kept, and cannot leave out one loop. A rule that skipped
a loop over a fixed seed would also skip most hand-written properties, which
fix their seed to replay a failure.

## Drawbacks

- **A second module in one repository.** Its tag starts with `lint/`, and
  `ergon` runs every stage once for each module. The repository commits a
  `go.work`, so every go command inside it runs in workspace mode.
- **The module requires `golang.org/x/tools`,** and follows its releases.
- **The host `go.dokimi.dev` needs a page for the module's path.**
- **40 of the 61 rules report without a fix,** so the author rewrites those
  checks by hand.
- **An origin is a call, and a block is one list of statements.** The
  analyzer misses a value that a plain assignment copies into a second
  variable. It also misses a check whose statements are in two blocks or
  two functions.
- **The command's test builds the command,** which adds a build of the
  command to the module's test run.

## References

| What | Where |
|---|---|
| `go/analysis`, `analysistest` and `singlechecker` | `golang.org/x/tools` v0.51.0 |
| The checker under `-fix` applies the fixes and prints no diagnostic | `go/analysis/internal/checker/checker.go` of `golang.org/x/tools` v0.51.0 |
| testifylint, the checkers `len`, `compares`, `empty`, `error-nil` and `bool-compare` | <https://github.com/Antonboom/testifylint> |
| `go-ruleguard` | <https://github.com/quasilyte/go-ruleguard> |
| Coverage of a program that a test runs, through `GOCOVERDIR` | <https://go.dev/doc/build-cover> |
| `go install` refuses a module whose `go.mod` has a `replace` directive | `go help install` in Go 1.27.1 |
| `go vet -fix` and `go fix -fixtool` | `go help vet` and `go help fix` in Go 1.27.1 |
