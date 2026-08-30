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

Every assertion exists as a function and, where its first argument is
the value being examined, as a chain method:

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
| `go.dokimi.dev/assert` | 34 assertions and a 15-method chain, stopping at the first failure |
| `go.dokimi.dev/assert/expect` | the same, recording and continuing |
| `go.dokimi.dev/assert/golden` | comparison against a recorded file, with scrubbers for content that changes each run |
| `go.dokimi.dev/assert/bench` | ceilings on latency, allocations and bytes per benchmark iteration |
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

## Assertion reference

### Values

| Name | What it states |
|---|---|
| `Equal` | Structural equality. A null collection does not equal an empty one, no type coercion, NaN is unequal to itself, floats compare exactly, cycles stop, functions compare by identity. |
| `NotEqual` | Negation of equal. |
| `True` | The condition holds. The failure carries the caller's message alone. |
| `False` | The condition does not hold. |
| `Nil` | The value is absent. A typed nil counts as nil. |
| `NotNil` | The value is present. A typed nil counts as nil. |
| `Length` | The container holds the stated number of items. Answers for any sized container or text. |
| `Empty` | The container holds nothing. |
| `NotEmpty` | The container holds something. |

### Text and containment

| Name | What it states |
|---|---|
| `Contains` | Text holds a substring, a sequence holds an element, or a map holds a key. |
| `NotContains` | Negation of contains. |
| `ContainsInOrder` | Text holds every needle, each after the previous one's match ends. |
| `HasPrefix` | Text starts with the given prefix. |
| `HasSuffix` | Text ends with the given suffix. |
| `Matches` | Text matches a regular expression. A pattern that does not compile is a failure, not an error. |

### Numbers and order

| Name | What it states |
|---|---|
| `CloseTo` | A number is within a tolerance of another, by absolute difference. NaN is outside every tolerance. |
| `InRange` | A number falls in a closed interval. NaN is in no range. |
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
| `CompletesWithin` | A subject finishes before the stated duration. |
| `Pure` | Observed state is unchanged across a call. |
| `NilContextSafe` | A subject given an absent cancellation handle does not crash. |

### Waiting

| Name | What it states |
|---|---|
| `Eventually` | An assertion body passes within a timeout, retried at an interval. Reports the last failure. |
| `EventuallyTrue` | A predicate becomes true within a timeout, retried with backoff. |
| `NoGoroutineLeaks` | No concurrent task started in the scope outlives it. |

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

### Proof

| Name | What it states |
|---|---|
| `Rejects` | A check fails against an implementation it is meant to reject. Yields the failure message. |

## The standard

The assertions are defined in `assert-spec`, language-neutral, and
implemented in six languages. This library vendors the definition and
holds itself to it on every run:

- **Completeness.** Every assertion is present under the name the
  definition gives it, or declared absent with a stated reason. An
  undeclared absence fails the build; so does a declared absence for
  something that is implemented.
- **Parity.** Both surfaces carry the same members.
- **Meaning.** 70 corpus cases state what an assertion must report,
  shared with every other implementation.

The corpus reaches 17 of the 41. A case states its arguments as data,
so it cannot cover an assertion that takes a callable, a cancellation
handle, a golden file or a benchmark. Those are checked for presence
and tested here.

## Development

```sh
make check      # the full pre-merge gate
make test       # tests
make lint       # vet, golangci-lint, markdown, licence headers, vulnerabilities
make spec-sync  # refresh the vendored definition
```

## Licence

MIT. See [LICENSE](LICENSE).
