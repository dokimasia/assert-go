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
	.Name: -"widget" +"wigdet"
```

## Install

```sh
go get go.dokimi.dev/assert
```

Requires Go 1.27. No dependency outside the standard library.

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
| A pointer, a map or a slice compares by the values it refers to | `ByIdentity()` compares it by identity |
| Floats compare exactly | `CloseTo` applies a tolerance |
| Unexported fields take part | — |
| No method of a value runs, so a type's `Equal` method does not decide | — |
| A map key compares as any value compares | — |
| Two functions are equal when they have the same code pointer, so two closures of one function literal are equal whatever they capture | — |
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

assert.Length(t, got, 1, "the check fails once")
assert.Equal(t, got[0].Contract, "the key was already present",
    "and fails for the reason the check is about")
```

Assert on the returned failure records, which `Rejects` returns in call
order. A subject that panics before the check's own assertion runs
satisfies a bare call.

## Packages

| Import | What it holds |
|---|---|
| `go.dokimi.dev/assert` | 50 assertions and a 15-method chain, stopping at the first failure |
| `go.dokimi.dev/assert/expect` | the same, recording and continuing |
| `go.dokimi.dev/assert/golden` | comparison against a recorded file or tree of files, with scrubbers for content that changes each run |
| `go.dokimi.dev/assert/files` | trees of files for a test, and the assertions on the files that code reads and writes |
| `go.dokimi.dev/assert/bench` | ceilings on latency, allocations and bytes per benchmark iteration |
| `go.dokimi.dev/assert/prop` | property checks over generated inputs, the generators, and the bridge to `go test -fuzz` |
| `go.dokimi.dev/assert/history` | the record of concurrent calls, the driver of the clients, and the checks that a record is linearizable, serializable, or has snapshot isolation |
| `go.dokimi.dev/assert/stateful` | machines that take the steps of a property's case against a model of their subject, and the task scheduler |
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

`golden.MatchTree` compares a tree of files, such as the directory that a
generator writes, with a golden directory under `testdata/golden`, and
`-update` rewrites the directory. It does not read the modes of the golden
tree, because a checkout writes modes by the umask of whoever checks it
out. It compares the execute bit of each file, the one bit that git
records.

## Files

`files.Workspace` writes a tree into a directory of the test's own. The
comparisons of trees read a directory through `os.DirFS`, or a tree in
memory through `fstest.MapFS`:

```go
dir := files.Workspace(t, files.Tree{
    "go.mod": files.Text("module example.com/a\n"),
    "a/a.go": files.Text("package a\n\nfunc Old() {}\n"),
})
assert.NoError(t, rename.Run(dir, "a.Old", "New"), "the rename succeeds")
files.Equal(t, os.DirFS(dir), files.Tree{
    "go.mod": files.Text("module example.com/a\n"),
    "a/a.go": files.Text("package a\n\nfunc New() {}\n"),
}, "the rename rewrites the declaration")
```

Where a tree states no mode, a workspace sets 0644 on a file and 0755 on
an executable file and on a directory, so it writes the same tree under
any umask. A comparison compares a mode only where the wanted entry states
one with `WithMode`, and otherwise the owner's execute bit of a file. It
follows no link: a link is an entry with its target. A failure lists the
first 64 paths that differ, and prints each entry as the `files`
expression that states it.

The assertions of one path, such as `files.HasMode`, check the entry at a
path of the operating system. `files.Read` returns the content of a file,
so the text assertions apply to it.

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

With `Warmup(n)`, the contract runs n iterations before it measures any,
and a cache that the body fills in its first iterations counts against no
ceiling. `RunParallel` takes the place of the loop for a body that runs on
`GOMAXPROCS` goroutines at once, as `testing.B.RunParallel` does. The
contract checks the ceilings of the reported run alone.

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
function that allocates on 99 of the 100 calls.

`MaxAllocsWithSetup` counts a function whose input a setup builds before
each call, such as a decoder that consumes its buffer. It counts the same
100 calls on one processor, and leaves every call of the setup out.

In a build with the race detector, msan or asan, in one whose
`-gcflags` turn off optimisation or inlining, and in a test binary that
a mutation run instrumented, the assertions and the contracts check no
allocation ceiling, because those builds allocate differently from a
production build. A mutation run sets `DOKIMI_MUTATE_INSTRUMENTED` in
every run of such a binary, and leaves it out of the ordinary build of
one mutant that confirms a survivor, where the ceilings apply.
`MaxAllocs` and `MaxAllocsWithSetup` still call the function, and a
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
runs the same body as `ForAll` does under `go test`, and as a fuzz target
under `go test -fuzz`.

`DOKIMI_ASSERT_PROP_PROFILE=campaign` runs every property as a campaign
for as long as `DOKIMI_ASSERT_PROP_BUDGET` states in seconds. A case that
counts a new label, records a new fingerprint or records a better score
with `Case.Target` joins the campaign's pool, and most later cases mutate
a member of the pool. The campaign stores each failure that it finds, and
reports every one when the budget has passed.

`prop.Hermetic()` makes a run read none of the `DOKIMI_ASSERT_PROP_*`
variables, for a test that runs a property to check something other than
a subject: a pin that compares a seeded run with a golden file, or a test
of a property harness that expects the property to fail. In a test binary
that a mutation run instrumented, which runs with `DOKIMI_MUTATE_MUTANT`
in its environment, every property derives its seed from its contract
and runs as an ordinary run, without writing to the store.

## Histories

A store, a queue or a cache that two or more clients use at once is
correct when every call appears to take effect at one instant inside its
own interval. A test records each call in a `history.History`, and checks the
history against a sequential model of the subject:

```go
func TestRegisterIsLinearizable(t *testing.T) {
    h := history.New()
    reg := NewRegister()
    history.Concurrently(2, time.Minute, func(client int) (any, error) {
        for i := range 50 {
            c := h.Invoke(client, "write", []any{client*1000 + i}, "x")
            reg.Write(client*1000 + i)
            c.OK(nil)
            c = h.Invoke(client, "read", nil, "x")
            c.OK(reg.Read())
        }
        return nil, nil
    })
    history.Linearizable(t, h, history.Model[int]{
        Init: func() int { return 0 },
        Step: func(s int, op history.Op) []int {
            if op.Operation == "write" {
                return []int{op.Args[0].(int)}
            }
            if !op.Known || op.Output == s {
                return []int{s}
            }
            return nil
        },
    }, "the register is linearizable")
}
```

The keys after the arguments of `Invoke` name what a call touches, and
the check searches the calls of each key on their own. A call whose
outcome is unknown, such as a timeout, completes with `Unknown`, and the
check lets it take effect at any later point, or never. The standard
fixes the search and its budget of 10,000,000 model steps per partition,
so one history gets one verdict in each implementation. A search that
uses up its budget fails as undecided, and the record states the limit
that stopped it.

`history.ModelFrom` builds the model from the subject itself, and then
the check finds a call that was not atomic. `history.FromIntervals`
builds a history from calls that a log recorded with a start and an end.

`history.Serializable` and `history.HasSnapshotIsolation` check a store
of transactions without a model. Each transaction is one call of the
operation `"txn"`, whose arguments are its micro-operations: an append of
a value to a key's list, or a read of a key's whole list. Every value is
appended to its key once:

```go
c := h.Invoke(client, "txn", []any{
    []any{"append", "x", 7},
    []any{"read", "y", nil},
}, "x", "y")
y, err := store.AppendThenRead(ctx, "x", 7, "y")
switch {
case errors.Is(err, ErrAborted):
    c.Fail(err)
case err != nil:
    c.Unknown(err)
default:
    c.OK([]any{[]any{"append", "x", 7}, []any{"read", "y", y}})
}
```

A check derives the dependencies between the committed transactions that
the reads reveal, and fails with the first anomaly that its level
forbids. Serializability forbids an inconsistent read, an aborted or
intermediate read, and every cycle of dependencies. Snapshot isolation
permits a cycle with two adjacent read-write dependencies, such as a
write skew. Both checks always decide. A record states the transactions,
the cycle and the evidence of each dependency.

## Machines

`stateful.Steps` takes the steps of a machine in a property's case. The
machine states a model of the subject and the actions that a step takes,
and every decision of the steps is a choice of the case, so a failing case
shrinks to the steps that the failure needs:

```go
func TestQueue(t *testing.T) {
    prop.ForAll(t, "the queue keeps its values in order", func(c *prop.Case) {
        q := NewQueue()
        written := 0
        stateful.Steps(c, stateful.Machine[[]int]{
            Model: history.Model[[]int]{Init: func() []int { return nil }, Step: queueStep},
            Actions: []stateful.Action[[]int]{{
                Name:  "put",
                Input: func(*prop.Case, []int) any { written++; return written },
                Run: func(c *prop.Case, client int, v any) {
                    call := c.History().Invoke(client, "put", []any{v})
                    q.Put(v.(int))
                    call.OK(nil)
                },
            }, {
                Name: "get",
                Run: func(c *prop.Case, client int, _ any) {
                    call := c.History().Invoke(client, "get", nil)
                    call.OK(q.Get())
                },
            }},
        }, stateful.Clients(2))
    })
}
```

After each step, `history.Linearizable` checks the case's history against
the model, with every call in one partition. With `Clients` of 2 or more,
a case lists the steps of a concurrent section and then runs them on its
clients at once. The clients run on threads by default, and each case
then runs up to four times. Under `stateful.Tasks`, they run as tasks of a
`stateful.Scheduler`, whose releases are choices of the case, so a race
replays and shrinks. A counterexample lists the steps among the values
that the case drew, and `prop.Draws` runs such a list, such as the steps
of a production incident, as the first case of a run.

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
| `Equal` | Structural equality. A null collection does not equal an empty one, no type coercion, NaN is unequal to itself, floats compare exactly and negative zero equals zero, cycles stop, functions compare by their code pointers. |
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
| `Contains` | Text has the substring, a sequence has an element equal to the needle, or a map has a key equal to the needle. An element and a key compare as `Equal` compares, so an int does not match a float. |
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
| `MaxAllocsWithSetup` | A callable makes at most a stated number of heap allocations per call on an input that a setup builds before each call. The setup is not counted. One setup and one call warm both first, and the count is the average over the calls after them, rounded down. |

### Golden files

| Name | What it states |
|---|---|
| `golden.Match` | Output matches the golden file resolved against the conventional directory. |
| `golden.MatchAt` | Output matches the golden file at a given path. |
| `golden.MatchJSONField` | Output matches one named field of a golden JSON object. |
| `golden.MatchTree` | A tree of files matches the golden tree resolved against the conventional directory. The comparison reads no mode of the golden tree, and compares execute bits. |

### Files

| Name | What it states |
|---|---|
| `files.Equal` | A tree read from a file system has the entries of a stated tree, no more and no fewer, each as the stated entry states it. |
| `files.Contains` | A tree read from a file system has every entry of a stated tree, each as the stated entry states it. It may have more. |
| `files.Unchanged` | A callable leaves the tree in a file system as it found it, modes included. |
| `files.Absent` | Nothing is at a path, a link whose target is missing included. |
| `files.IsFile` | A file is at a path. A link to a file is a link. |
| `files.IsDir` | A directory is at a path. A link to a directory is a link. |
| `files.LinksTo` | A symbolic link with a stated target is at a path. |
| `files.HasContent` | A file whose bytes equal stated text or bytes is at a path. |
| `files.HasMode` | A file or a directory whose nine permission bits equal a stated mode is at a path. |

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

### Histories

| Name | What it states |
|---|---|
| `history.Linearizable` | Every partition of a recorded history has an order of its calls that keeps the history's precedence and that the model accepts. A search that uses up its budget or its memo limit is undecided, and fails. |
| `history.Serializable` | The list-append transactions of a history exhibit no anomaly that serializability forbids, among the dependencies that their reads reveal. |
| `history.HasSnapshotIsolation` | The list-append transactions of a history exhibit no anomaly that snapshot isolation forbids: every cycle of their dependencies has two adjacent read-write dependencies. |

### Proof

| Name | What it states |
|---|---|
| `Rejects` | A check fails against an implementation it is meant to reject. Yields the failure records of the check, in call order. |

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
- **Meaning.** 195 corpus cases state what an assertion must report,
  shared with every other implementation. Each case runs through every
  function and chain form of both surfaces. The record of a failing
  case must state the case's assertion and the message unchanged, and
  contain exactly the fields that the assertion declares. The call
  record of each case must state the assertion, the message, the
  verdict and the surface, and on a failure the same fields as typed
  literals.
- **Properties.** 430 vectors state how the property engine decodes,
  generates and shrinks inputs, decides coverage, encodes replay tokens,
  stores failures, reports a run and records its calls, shared with
  every other implementation.
- **Histories.** 80 vectors state the events that a history records, the
  history that `FromIntervals` builds from a log, the verdict, the steps
  and the record of each check of a history against a named model, and
  the verdict and the record of each isolation check of a history of
  transactions, shared with every other implementation.
- **Machines.** 14 vectors state the steps that seven machine subjects
  take, the counterexample that a failure shrinks to, a race that the
  task scheduler finds, the traces that a run follows or refuses, and the
  step that a flaky run's divergence names, shared with every other
  implementation.
- **Files.** 57 vectors state the verdict and the record of each
  assertion that reads files over a workspace, and the golden tree that
  an update leaves, shared with every other implementation.

A corpus case states its arguments as data, or names a behaviour that
each implementation builds, such as a callable that panics. The cases
cover 39 of the 70 assertions outside `prop`. No case can state an error
value, a golden file, a benchmark, a predicate or a history, so those
assertions are checked for presence and tested here. The history vectors
cover `linearizable`, `serializable` and `snapshot-isolation`, and the
files vectors cover the ten assertions that read files.
The vectors cover 38 of the 40 property assertions: `prop-for-all` and
every property form but `prop-max-allocs` and
`prop-max-allocs-with-setup`, whose allocation counts no vector can state.

## Development

```sh
make check      # the full pre-merge gate
make test       # tests
make lint       # vet, golangci-lint, markdown, licence headers, vulnerabilities
make spec-sync  # refresh the vendored definition
```

## Licence

MIT. See [LICENSE](LICENSE).
