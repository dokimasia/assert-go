# Migrating

## A nil collection no longer equals an empty one

Comparisons here do not apply `cmpopts.EquateEmpty()`. A nil slice does
not equal an empty slice, and a nil map does not equal an empty map.

A test that passed where the comparison equated them:

```go
Equal(t, got, []int{}, "returns no items")   // got is nil: passed
```

fails here:

```go
assert.Equal(t, got, []int{}, "returns no items")   // got is nil: fails
```

The reason is not Go. The same assertions exist in Python, PHP and
TypeScript, where `None`, `null` and `undefined` are values a test may
need to tell apart from an empty list. Equating them everywhere would
make those libraries report values they were never given.

Two ways forward. Prefer the first.

Assert what the code returns. If the subject returns nil, say nil:

```go
var want []int
assert.Equal(t, got, want, "returns no items")
```

Or relax it on the call, where the difference does not matter:

```go
assert.Equal(t, got, []int{}, "returns no items", assert.EquateEmpty())
```

The option applies to that call and to nothing else.

## Names

| Was | Now |
|---|---|
| `Error` | `HasError` |
| `Sequence` | `Pairwise` |
| `AllocsMax`, `BytesMax`, `LatencyMax`, `MeanMax` | `MaxAllocs`, `MaxBytes`, `MaxLatency`, `MaxMean` |
| `FailableTB` | `Recorder` |
| `Assert(tb, got)` | `That(tb, got)` |
| `Equals`, `HasLen`, `IsNil`, `IsEmpty` | `Equal`, `Length`, `Nil`, `Empty` |

One name per assertion now, used by both the function and the chain
form. The two surfaces previously disagreed: `Equal` against `Equals`,
`Len` against `HasLen`.

## Assertions that went away

`AssertBounded` was a range check over a function result. Call `InRange`
on the result.

`AssertNilSafe` was "does not panic" with a nil argument. Call
`NotPanics`.

## The seat gained a method

`TB` requires `Errorf` alongside `Helper` and `Fatalf`, so the
recording surface has somewhere to report. `testing.T` and `testing.B`
are unaffected. A hand-written seat needs one more method.
