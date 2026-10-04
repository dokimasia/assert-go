# assert

Test assertions for Go, defined by a language-neutral standard and held
to it on every run.

```go
import "go.dokimi.dev/assert"

func TestGet(t *testing.T) {
    item, err := store.Get(ctx, id)

    assert.NoError(t, err, "Get succeeds for a key that is present")
    assert.Equal(t, item, want, "Get returns the stored item")
}
```

Every assertion takes a message last. It states the contract under
test and is the first line of the failure, so a failure says what was
supposed to be true rather than only what was observed:

```text
Get returns the stored item: (-want +got)
  store.Item{
  	ID:   "abc",
- 	Name: "widget",
+ 	Name: "wigdet",
  }
```

## Install

```sh
go get go.dokimi.dev/assert
```

Requires Go 1.27. One dependency: `github.com/google/go-cmp`.

## Two surfaces

`assert` stops the test at the first failure. `expect` records the
failure and carries on, for when several properties of one value are
each worth seeing:

```go
expect.That(t, user).
    NotNil("the user was found").
    HasPrefix("usr_", "the id carries its prefix").
    Length(3, "every field was populated")
```

One run reports all three. Both packages carry the same assertions
under the same names and share one comparison; a conformance test
fails the build if they ever diverge.

Every assertion exists as a function. The fifteen that examine a value
of any type, such as `Equal` and `Contains`, also exist as chain
methods:

```go
assert.Equal(t, got, want, "the values match")
assert.That(t, got).Equal(want, "the values match")
```

## Equality

The rules most libraries leave to their comparison library's defaults,
stated instead:

| Rule | Reverse it with |
|---|---|
| A nil map or slice does not equal an empty one | `EquateEmpty()` |
| NaN does not equal NaN | `EquateNaNs()` |
| Floats compare exactly | `CloseTo` applies a tolerance |
| Unexported fields take part | — |
| Values of different types never compare | — |

The first is the one that catches people. `[]int(nil)` and `[]int{}`
are different answers, and a test may need to tell them apart. The
same assertions exist in Python, PHP and TypeScript, where `None`,
`null` and `undefined` are distinct from an empty list; equating them
everywhere would make those libraries report values they were never
given.

An option applies to the call it is passed to and to nothing else.

## Testing your own assertions

A check whose every statement is `NoError` passes against a subject
whose methods do nothing and return nil. It reads as coverage and
establishes nothing.

`Rejects` drives a check against an implementation it is meant to
reject, and fails when the check passes:

```go
got := assert.Rejects(t, "a store that overwrites fails the check",
    func(tb assert.TB) { refusesADuplicate(tb, overwritingStore{}) })

assert.Contains(t, got, "the key was already present",
    "and fails for the reason the check is about")
```

Assert on the returned message. A subject that panics before reaching
the assertion satisfies a bare call while the check's own assertion
never ran.

## Packages

| Import | What it holds |
|---|---|
| `go.dokimi.dev/assert` | 49 assertions and a 15-method chain, stopping at the first failure |
| `go.dokimi.dev/assert/expect` | the same, recording and continuing |
| `go.dokimi.dev/assert/golden` | comparison against a recorded file, with scrubbers for content that changes each run |
| `go.dokimi.dev/assert/bench` | ceilings on latency, allocations and bytes per benchmark iteration |
| `go.dokimi.dev/assert/prop` | property checks over generated inputs, the generators, and the bridge to `go test -fuzz` |
| `go.dokimi.dev/assert/conformance` | this library checked against the standard |

## Golden files

```go
golden.Match(t, "response.json", body, golden.ShouldUpdate(),
    golden.ScrubTimestamps())
```

`go test -update` rewrites the file. Read the diff before you do: a
golden file updated without reading it records whatever the code now
does, which is the opposite of an assertion.

Scrubbers replace content that differs between runs, on both sides of
the comparison, so the parts that should be stable are the parts
compared. `ScrubTimestamps`, `ScrubHashes`, `ScrubRunIDs` and
`ScrubJSONFields` are supplied; a `Scrubber` is a `func(string) string`.

## Benchmark ceilings

A benchmark records numbers; somebody has to read them to notice a
regression. A contract states the ceiling in the benchmark, so
exceeding it fails the build:

```go
func BenchmarkGet(b *testing.B) {
    c := bench.Start(b).MaxLatency(50 * time.Microsecond).MaxAllocs(2)
    defer c.End()

    for c.Loop() {
        _, _ = store.Get(ctx, id)
    }
}
```

Ceilings are checked together, so one run names each one exceeded. The
p99 rather than the mean, because the tail is what a caller waits for.

A contract is checked only when benchmarks run. `MaxAllocs` states an
allocation ceiling in a test, so the ordinary test run checks it:

```go
func TestGetAllocs(t *testing.T) {
    assert.MaxAllocs(t, func() { _, _ = store.Get(ctx, id) }, 0,
        "Get averages under one allocation per call once the store is warm")
}
```

It calls the function once to warm it and counts the next 100 calls,
through `testing.AllocsPerRun`, so the test that calls it does not call
`t.Parallel`. The average is rounded down. A ceiling of 0 then passes a
function that allocates on 99 of the 100 calls. In a build with the race
detector, msan or asan, and in one whose `-gcflags` turn off
optimisation or inlining, neither form
checks an allocation ceiling, because those builds allocate differently
from the one that ships. `MaxAllocs` still calls the function, and a
contract still publishes its counts.

## Properties

`prop.ForAll` runs a body against generated inputs. When a case fails,
it shrinks the case to the smallest one that fails the same way and
reports it:

```go
func TestRoundTrip(t *testing.T) {
    prop.ForAll(t, "decoding undoes encoding", func(c *prop.Case) {
        v := c.Draw(prop.List(prop.Integer[int64](-1000, 1000), prop.MaxSize(16)), "values")
        got, err := Decode(Encode(v))
        assert.NoError(c, err, "decoding succeeds")
        assert.Equal(c, got, v, "decoding returns the encoded values")
    })
}
```

The standard fixes the random source, the decoding of each generator,
the phases of a run and the shrink passes. A seed gives the same inputs
in each implementation of the standard, and a failure shrinks to the same
counterexample in each. Every assertion works in the body, because the
case is a seat.

A failure states the counterexample, the seed and a replay token.
`prop.Replay` and `DOKIMI_ASSERT_PROP_REPLAY` run the token's case again.
The run also writes the smallest failing case of each failure to
`testdata/prop/<test name>/` beside `testdata/golden`. The next run tries
that case first. Review the store as you review golden files. `prop.Fuzz`
runs the same body under `go test -fuzz`.

## Call records

Set `DOKIMI_ASSERT_RECORD=1` to record every assertion call of a run:

```sh
DOKIMI_ASSERT_RECORD=1 go test -json ./...
```

Each call writes one call record, a JSON object, as an `attr` event of
its test under the key `dokimi.assert.<seq>`. A call record states:

- The call's number in its test.
- The assertion, the message, and the verdict: `pass`, `fail` or
  `error`.
- Whether the surface stops the test, and the file and the line of the
  call.
- The detail of a failure, each value a typed literal of the standard.

A call in the body of a property, of `Eventually` or of `Rejects` states
the call that ran the body. A call in a property's case also states the
case's phase, such as `random` or `shrink`. A property's own record
states the detail of its run on a pass too. A record longer than one
line of `go test -json` is split over events of the same key, in order.

Recording is off by default, and an unset, empty or `0` value turns it
off. Any other value is an error that every assertion reports. An
`assert.Recorder` keeps the call records of its own calls whatever the
variable states, and `Records` returns them.

## Assertion reference

### Values

| Name | What it states |
|---|---|
| `Equal` | Structural equality. A null collection does not equal an empty one, no type coercion, NaN is unequal to itself, floats compare exactly and negative zero equals zero, cycles stop, functions compare by identity. |
| `NotEqual` | Negation of equal. |
| `True` | The condition holds. The failure carries the caller's message alone. |
| `False` | The condition does not hold. |
| `Nil` | The value is absent. A typed nil counts as nil. |
| `NotNil` | The value is present. A typed nil counts as nil. |
| `Length` | The container has the stated number of items: the elements of a slice, array or channel, the entries of a map, the bytes of a `[]byte`, or the Unicode scalar values of a string. A value without a length, nil included, fails. |
| `Empty` | The container has no item. A value without a length, nil included, fails. |
| `NotEmpty` | The container has an item. A value without a length, nil included, fails. |

### Text and containment

| Name | What it states |
|---|---|
| `Contains` | Text has the substring, a sequence has an element equal to the needle, or a map has the key. An element compares as `Equal` compares, so an int does not match a float. |
| `NotContains` | Negation of contains. |
| `ContainsInOrder` | Text holds every needle, each after the previous one's match ends. |
| `Permutation` | A slice contains the same elements as another, each as often, in any order. Elements compare as `Equal` compares them. |
| `HasPrefix` | Text starts with the given prefix. |
| `HasSuffix` | Text ends with the given suffix. |
| `Matches` | Text matches a pattern of the portable subset, which every implementation reads the same way: `$` matches at the end of the text only, `\d`, `\w` and `\s` are ASCII classes, and `.` matches no line terminator. A pattern outside the subset is a failure, not an error. |

### Numbers and order

| Name | What it states |
|---|---|
| `CloseTo` | A number is within a tolerance of another, by absolute difference. NaN is outside every tolerance, and equal infinities are not close. |
| `InRange` | A number falls in a closed interval. NaN is in no range, and a range with a NaN bound contains no number. |
| `Pairwise` | Every adjacent pair of a sequence satisfies a predicate. Nought or one item passes. |

### Errors and panics

| Name | What it states |
|---|---|
| `NoError` | No failure value is present. |
| `HasError` | Some failure value is present. |
| `ErrorIs` | A failure matches a sentinel, through the chain of wrapped causes. |
| `ErrorIsNot` | A failure does not match a sentinel. |
| `ErrorAs` | A failure of the given type is in the chain. Yields it. |
| `Panics` | A callable raises. Yields what was raised. |
| `NotPanics` | A callable does not raise. |

### Behaviour

| Name | What it states |
|---|---|
| `HonoursCancellation` | A subject given a cancelled handle reports a cancellation failure. |
| `HonoursDeadline` | A subject given an expired deadline reports a deadline failure. |
| `CompletesWithin` | A subject finishes before the stated duration. A subject still running when the duration passes fails then. |
| `Pure` | Observed state is unchanged across a call. |
| `NilContextSafe` | A subject given an absent cancellation handle does not crash. |

### Relations

A relation calls its subject and relates the runs to each other, so a
test needs no expected output. An error or a panic of a callable fails
the relation, except where the relation requires a failure.

| Name | What it states |
|---|---|
| `Idempotent` | Calling a subject twice with one input leaves the observed state that calling it once left. |
| `Accumulates` | Each call of a subject with one input changes an observed integer by the same amount, and the first call changes it. |
| `Deterministic` | 32 calls of a subject with one input return equal results. |
| `Commutative` | Combining a and b gives what combining b and a gives. |
| `Associative` | Combining the combination of a and b with c gives what combining a with the combination of b and c gives. |
| `RoundTrip` | The inverse conversion of the forward conversion of an input returns the input. |
| `StableOrder` | 32 iterations of a subject yield equal sequences. |
| `NoDuplicates` | One iteration of a subject yields each element at most once. |
| `Monotonic` | An observed value never falls and is never NaN while a subject advances a stated number of steps. |
| `Total` | A call succeeds for every input of a domain, in order. |
| `NotPure` | Observed state changes across a call. |
| `FailsAfterClose` | After a subject closes, a call fails with a sentinel, through the chain of wrapped causes. |
| `Poisoned` | After a failure is induced, 32 readings each report a failure. |

### Waiting

| Name | What it states |
|---|---|
| `Eventually` | An assertion body passes within a timeout, retried at an interval. Reports the last failure. |
| `EventuallyTrue` | A predicate becomes true within a timeout, retried with backoff. |
| `NoGoroutineLeaks` | No concurrent task started in the scope outlives it. |

### Allocations

| Name | What it states |
|---|---|
| `MaxAllocs` | A callable makes at most a stated number of heap allocations per call. One call warms it first, and the count is the average over the calls after it, rounded down. |

### Golden files

| Name | What it states |
|---|---|
| `golden.Match` | Output matches the golden file resolved against the conventional directory. |
| `golden.MatchAt` | Output matches the golden file at a given path. |
| `golden.MatchJSONField` | Output matches one named field of a golden JSON object. |

### Benchmarks

| Name | What it states |
|---|---|
| `bench.Contract.MaxLatency` | The p99 latency per iteration stays within a ceiling. |
| `bench.Contract.MaxMean` | The mean latency per iteration stays within a ceiling. |
| `bench.Contract.MaxAllocs` | The allocations per iteration stay within a ceiling. |
| `bench.Contract.MaxBytes` | The bytes allocated per iteration stay within a ceiling. |

### Properties

| Name | What it states |
|---|---|
| `prop.ForAll` | A body passes for every input a run generates. A failure states the smallest counterexample that the shrink passes find. |
| `prop.Fuzz` | A body passes for every input a fuzzer finds, decoded into a case by the bridge rules. |

### Proof

| Name | What it states |
|---|---|
| `Rejects` | A check fails against an implementation it is meant to reject. Yields the failure message. |

## The standard

The assertions are defined in `assert-spec`, language-neutral, and
implemented in six languages. This library vendors the definition and
holds itself to it on every run:

- **Completeness.** Every assertion is present under the name the
  definition gives it, or declared absent with a stated reason. The
  build fails on an undeclared absence, and on a declared absence of
  something that is implemented. Each function and method takes the
  number of arguments that the definition states. `NoGoroutineLeaks` is
  the one exception: it returns its check instead of taking the scope.
- **Parity.** Both surfaces carry the same members.
- **Meaning.** 170 corpus cases state what an assertion must report,
  shared with every other implementation. Each case runs through every
  function and chain form of both surfaces. The record of a failing
  case must state the case's assertion and the message unchanged, and
  contain exactly the fields that the assertion declares. The call
  record of each case must state the assertion, the message, the
  verdict and the surface, and on a failure the same fields as typed
  literals.
- **Properties.** 419 vectors state how the property engine decodes,
  generates and shrinks inputs, decides coverage, encodes replay tokens,
  stores failures, reports a run and records its calls, shared with
  every other implementation.

A corpus case states its arguments as data, or names a behaviour that
each implementation builds, such as a callable that panics. The cases
cover 39 of the 56 assertions outside `prop`. No case can state an error
value, a golden file, a benchmark or a predicate, so those assertions are
checked for presence and tested here. The vectors cover 38 of the 39
property assertions: `prop-for-all` and every property form but
`prop-max-allocs`, whose allocation count no vector can state.

## Development

```sh
make check      # the full pre-merge gate
make test       # tests
make lint       # vet, golangci-lint, markdown, licence headers, vulnerabilities
make spec-sync  # refresh the vendored definition
```

## Licence

MIT. See [LICENSE](LICENSE).
