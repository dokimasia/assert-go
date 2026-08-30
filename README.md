# assert

Test assertions for Go, defined by a language-neutral standard and
checked against it.

```go
assert.Equal(t, store.Get(ctx, id), item, "Get returns the stored item")
assert.ErrorIs(t, err, store.ErrNotFound, "a missing key is reported as missing")

assert.That(t, resp.StatusCode).
    NotEqual(0, "the status was set").
    Equal(200, "the request succeeded")
```

Every assertion takes a message last. It states the contract under
test, and it is the first line of the failure, so a failure says what
was supposed to be true rather than only what was observed.

## Two surfaces

`assert` stops the test at the first failure. `expect` records the
failure and carries on, which is what you want when several properties
of one value are each worth seeing:

```go
expect.That(t, user).
    NotNil("the user was found").
    HasPrefix("usr_", "the id carries its prefix").
    Length(3, "every field was populated")
```

Both carry the same assertions under the same names, and share one
comparison. A test enforces that they do.

## Equality

The rules most libraries leave to their comparison library's defaults,
stated instead:

- A nil map or slice does not equal an empty one. Pass `EquateEmpty()`
  where that difference does not matter.
- NaN does not equal NaN. Pass `EquateNaNs()` to reverse it.
- Floats compare exactly. `CloseTo` is the one that applies a tolerance.
- Unexported fields take part.
- Values of different types never compare; the type parameter refuses
  them at compile time.

## Packages

| Package | What it holds |
|---|---|
| `assert` | 34 assertions and a 15-method chain, stopping at the first failure |
| `expect` | the same, recording and continuing |
| `golden` | comparison against a recorded file, with scrubbers for content that changes each run |
| `bench` | ceilings on latency, allocations and bytes per benchmark iteration |
| `conformance` | this library checked against the standard |

## Proving a check can fail

A check whose every statement is `NoError` passes against a subject
whose methods do nothing. `Rejects` drives a check against an
implementation it is meant to reject, and fails when the check passes:

```go
got := assert.Rejects(t, "a store that overwrites fails the check",
    func(tb assert.TB) { refusesADuplicate(tb, overwritingStore{}) })

assert.Contains(t, got, "the key was already present",
    "and fails for the reason the check is about")
```

## The standard

The assertions are defined in `assert-spec`, language-neutral, and
implemented in six languages. This library vendors the definition and
holds itself to it on every run:

- Every assertion is present under the name the definition gives it,
  or declared absent with a reason.
- Both surfaces carry the same members.
- 70 corpus cases state what an assertion must report, and are shared
  with every other implementation.

An assertion whose arguments are not data — a callable, a cancellation
handle, a golden file — cannot be a corpus case. Those are checked for
presence and tested here.

## Dependencies

One: `github.com/google/go-cmp`, for the structural diff.

## Development

```
make check      # the full pre-merge gate
make test       # tests
make lint       # vet, golangci-lint, markdown, licence, vulnerabilities
make spec-sync  # refresh the vendored definition from ../assert-spec
```

## Licence

See [LICENSE](LICENSE).
