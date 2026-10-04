---
rfc: 0009
title: The history seam and the linearizability checker in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-04
updated: 2026-10-05
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0009: The history seam and the linearizability checker in Go

## Summary

`go.dokimi.dev/assert/history` implements the observation seams and the
linearizability checker that version 3.1.0 of the definition adds: the
history, `from-intervals`, the concurrency driver, the model,
`model-from`, the four options and the assertion `linearizable`. The
definition fixes the recording order, the search, its budget and its
record, and pins them with 42 vectors. This library translates the
executable reference in `tools/history` into Go, and the conformance
package runs every vector on every test run. The module gains no
dependency.

The design makes four decisions that the definition leaves to Go:

- **One public package.** The search is unexported code of `history`.
  Every vector runs it through `Linearizable`, the function that every
  caller runs.
- **A model is a struct of functions over a state type.** The defaults
  compare a state of type `bool`, `string`, or a predeclared integer or
  float type with Go's `==` and hash its value directly. For a state of
  any other type, they call `equality.Equal` and a new `equality.Hash`
  that agrees with it. A model that states its own `Equal` also states a
  `Hash` that agrees with it. Without one, the memo groups
  configurations by their calls alone.
- **A search keeps its states and its configurations in flat arrays.**
  One store contains the states of every configuration. A configuration
  is a record of 32 bytes, and the memo maps each key to the index of a
  record. The search stores a step's states only for a configuration
  that the memo lacks.
- **One mutex orders the history.** A usage error panics. `Concurrently`
  recovers each body's panic and raises the panic of the lowest client
  again on the caller's goroutine.

The RFC amends the list of public packages in `docs/standards/code.md`,
and the rules on panics and on `recover` in `docs/standards/errors.md`.

## Motivation

The definition states what the seam and the checker do. Go has to settle
how:

- **Which package contains the search, and how a vector observes it.**
  A vector pins the steps of a check that passes. A passing check
  reports no record, so no public value states its steps.
- **How the memo finds a state.** Go has no hash of an arbitrary value.
  The definition's RFC-0004 measured a memo that groups configurations
  by their calls alone: on the synthetic queue histories it ran for more
  than ten minutes on one row of 10 histories, and a memo that also
  hashes the state checked all 28 rows in 8.4 seconds.
- **What a step costs.** A check that spends the default budget calls
  `Step` 10,000,000 times, and compares and hashes the states of each
  call. Each 10 ns that a step costs adds 0.1 seconds to such a check.
- **How a model reads in a Go test.** The definition leaves the form of a
  model free. Its states can be of any type, slices and maps included.
- **How a usage error and a client's panic end.** The standards allow a
  panic in a constructor and a registration alone, and a `recover` only
  in an assertion about panics and at the end of a property's case. The
  definition requires the history to panic on a usage error, and the
  driver to collect every client's outcome before it raises a panic
  again.
- **The names that the naming table does not fix**, such as the type of
  an interval, the kind of an event and the types of the record's
  detail.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| History | `history`, new | Records invocations and completions in one order under one mutex, assigns processes, and returns its events without a gap |
| Intervals | `history` | Builds a history from calls recorded with a start and an end on one clock |
| Driver | `history` | Starts the clients of `Concurrently`, releases them together, waits up to the deadline, and returns their outcomes |
| Model | `history` | The model of a check, its defaults, and `ModelFrom` |
| Checker | `history`, unexported | The calls, the partitions, the search, the memo, the limits, the workers and the frontier |
| Record | `history` | The detail of a failing check, its JSON form, and the sentence that the writer renders |
| State hash | `internal/equality`, `Hash`, new | A hash of a value that agrees with `Equal` under no relaxation |
| Vectors | `conformance` | The seam and checker vectors, and the six named models |

`history` imports the root package for `TB`, the standard library, and
the internal packages `equality`, `fault`, `literal`, `matcher` and
`text`. `conformance` imports `history`. No package of the module imports
`history` but `conformance` and the tests.

### The public surface

```go
package history

// History is the events of one history, in one recording order that
// every client shares.
type History struct{ /* unexported: a mutex, the events, the open calls, the processes */ }

func New() *History
func (h *History) Invoke(client int, operation string, args []any, keys ...any) Call
func (h *History) Events() []Event

// Call is an open call, through which its client records the completion.
type Call struct{ /* unexported: the history and the index of the invocation */ }

func (c Call) OK(output any)
func (c Call) Fail(err error)
func (c Call) Unknown(err error)

// Event is one recorded event.
type Event struct {
	Index     int
	Kind      Kind
	Call      int
	Client    int
	Process   int
	Operation string // on an invocation
	Args      []any  // on an invocation
	Keys      []any  // on an invocation
	Output    any    // on an ok completion
	Error     error  // on a fail or an unknown completion
}

// Kind is the kind of an event. The zero Kind is no kind.
type Kind uint8

const (
	Invoke  Kind = 1 // invoke
	OK      Kind = 2 // ok
	Fail    Kind = 3 // fail
	Unknown Kind = 4 // unknown
)

// Interval is a call recorded with a start and an end on one clock.
type Interval struct {
	Client     int
	Operation  string
	Args       []any
	Keys       []any
	Start, End int64
	Kind       Kind  // OK, Fail or Unknown, or Invoke for a pending call, whose End is not read
	Output     any   // for OK
	Error      error // for Fail and Unknown
}

func FromIntervals(entries []Interval) (*History, error)

var ErrInterval = errors.New("history: an entry that from-intervals refuses")

func Concurrently(clients int, within time.Duration, body func(client int) (any, error)) []Outcome

// Outcome is what one client of Concurrently did.
type Outcome struct {
	Client   int
	Finished bool
	Output   any
	Error    error
}

// Model is a sequential model: the state before any call, and the states
// that a call may leave.
type Model[S any] struct {
	Init  func() S
	Step  func(state S, op Op) []S
	Equal func(a, b S) bool    // nil compares as assert.Equal compares
	Hash  func(state S) uint64 // nil hashes to agree with a nil Equal
}

// Op is a call as a model sees it.
type Op struct {
	Operation string
	Args      []any
	Known     bool
	Output    any
}

// Subject applies an operation to its arguments and returns the output.
type Subject func(operation string, args []any) any

func ModelFrom(factory func() Subject) Model[[]Op]

func Linearizable[S any](tb assert.TB, h *History, m Model[S], contract string, opts ...Option)

var ErrModel = errors.New("history: a function of a model panics or ends its goroutine")

// Option configures one check.
type Option struct{ /* unexported */ }

func Budget(steps int) Option
func MemoLimit(bits int64) Option
func TimeLimit(d time.Duration) Option
func Workers(n int) Option

// The detail of a failing check.
type Verdict uint8 // Passed, Violated, Undecided
type Limit uint8   // LimitSteps, LimitMemo, LimitTime
type Span struct {
	Call       int
	Completion int // -1 for a pending call
	Process    int
	Op         Op
}
```

The naming table fixes `History`, `New`, `Invoke`, `Events`, `Call` with
`OK`, `Fail` and `Unknown`, `Event`, `FromIntervals`, `Concurrently`,
`Outcome`, `Model` with `Init`, `Step` and `Equal`, `Op`, `ModelFrom`,
`Linearizable`, `Budget`, `MemoLimit`, `TimeLimit` and `Workers`. This
RFC decides the other names:

| Name | Why Go needs it |
|---|---|
| `Kind` and its four values | An event's kind is an enumeration, as `prop.Outcome` is |
| `Interval` | The entries of `from-intervals` are values of one type |
| `ErrInterval`, `ErrModel` | A public package exports the sentinel of each fault that a caller tests |
| `Model.Hash` | The memo needs a hash that agrees with a stated `Equal` |
| `Subject` | The factory of `ModelFrom` returns a function, and a named type reads better in its signature |
| `Option` | The four options are values of one type, as `prop.Option` is |
| `Verdict`, `Limit`, `Span` | The detail of a failing check states Go values, as a property's record does |

`Call` is a value of two words. `Invoke` allocates no handle for it. The
`outcome` field of a check's record is a `Verdict`, because the naming
table gives the name `Outcome` to the driver's result. The table also
gives `Call` to the open call. A call in a record is a `Span`: the call
from its invocation to its completion.

### The history

`Invoke` and each completion append their event at the next index under
the history's mutex. The mutex is the definition's sequentially
consistent counter. A client records the invocation before
its call to the subject starts and the completion after the call
returns. Call a precedes call b in the history only when a's completion
took the mutex before b's invocation did, so a returned before b
started. `Events` copies the events under the mutex, so a reading while
clients record has no gap.

The history keeps the values it receives and does not copy them. Each
client starts on a process of its own, numbered in the order of first
invocation. A completion as `Unknown` moves the client to a new process
for its next invocation.

`Invoke` encodes each key as a typed literal with `literal.Encode`, and
keeps the literal's text as the key's identity. Two keys are one key
when their texts are equal, so the int 1 and the float 1.0 are two keys,
and the ints 1 and `int64(1)` are one.

A usage error panics, and its message names the call:

| Usage error | Panic message |
|---|---|
| `Invoke` for a client whose call is open | `history: Invoke(0, "read") while call 4 of client 0 is open` |
| A second completion of one call | `history: call 4 completes a second time` |
| A key that no typed literal states | `history: Invoke(0, "read") states the key chan int, which no typed literal states` |
| `Concurrently` with fewer than one client | `history: Concurrently(0, 1m0s) starts no client` |
| `Concurrently` with a negative time | `history: Concurrently(2, -1s) waits a negative time` |

A history stores two events per call. The ten fields of an `Event` take
136 bytes on a 64-bit platform, as `unsafe.Sizeof` measures them, and
the values that its slices and interfaces point to take more.

`Event` implements `json.Marshaler` with the history's JSON form, which
the definition fixes: `args`, `keys` and `output` as typed literals, and
`error` as the error's text. A value that no typed literal states is an
opaque literal, as a call record states one.

### Histories recorded elsewhere

`FromIntervals` orders the invocations and completions of the entries by
time. At an equal time, an invocation comes before a completion, and the
invocations, or the completions, keep the order of their entries. It
then records them through a history, so the process rule applies.

An entry whose `Kind` is `Invoke` is pending: it has no completion, and
`FromIntervals` does not read its `End`. Each interval is closed, so two
entries that share an instant overlap, and a pending entry overlaps every
later entry of its client.

`FromIntervals` returns a fault of the kind `ErrInterval`, with the
operation `history.FromIntervals` and the path of the entry, for the
first entry in the given order that breaks a rule:

- It ends before it starts.
- It overlaps an earlier entry of its client.
- Its `Kind` is no kind.

The third rule is Go's own. A zero `Kind` is an entry that states no
completion, which a log reader produces when it forgets the field, and
reading it as pending would hide that mistake.

### The driver

`Concurrently` starts one goroutine per client. Each goroutine waits on a
release channel, and the driver closes the channel once it has started
every goroutine. The driver then waits until every body has ended or
`within` has passed on the platform clock, counted from the release. It
returns one `Outcome` per client, in client order.

| A body | Its outcome |
|---|---|
| Returns before `within` passes | `Finished`, with its output and its error |
| Is still running when `within` passes | Not `Finished`. Its goroutine runs on, and its open call is pending in the history |
| Ends its goroutine with `runtime.Goexit`, as `t.FailNow` does | `Finished`, with no output and no error |
| Panics | Recovered. After the wait, the driver raises again the value of the lowest-numbered client that panicked |

A body that ends after `within` passed writes nothing into the returned
outcomes, and a panic of such a body is recovered and dropped, so it
cannot end the test process. `NoGoroutineLeaks` reports the goroutine of
a body that runs on.

The driver raises the panic's own value, so a `recover` in the caller
sees what the body raised. The stack of the body's goroutine is not
kept.

### The model

A model is a struct of functions over a state type `S`, as Porcupine's
model is a struct of functions:

```go
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
```

| Field | When nil |
|---|---|
| `Init`, `Step` | A fault ends the check: the model states no initial state or no step |
| `Equal` | States compare as `assert.Equal` compares them. A state whose type is `bool`, `string`, or a predeclared integer or float type compares with Go's `==`, which agrees with `assert.Equal` on those types: -0 equals +0, and a NaN equals nothing. A state of any other type compares through `equality.Equal` with no relaxation |
| `Hash` | With a nil `Equal`, a state of those types hashes directly, and a state of any other type through `equality.Hash`. With a stated `Equal`, every state hashes alike, and the memo groups configurations by their calls alone |

The direct hash of an integer mixes its bits with the final mix of
MurmurHash3. A float adds 0 before the mix. The addition turns -0 into
+0, so the two zeros hash alike. A `bool` hashes as the integer 1 or 0
does. A string hashes with `hash/maphash` under a seed that each process
makes. No output of a check depends on the seed, because the search uses
a hash only to choose which states to compare.

The search passes the defaults pointers to the states in its arrays. A
comparison through reflection then copies no state to the heap.

A stated `Hash` agrees with the model's equality: two states that it
reports equal have one hash. Porcupine v1.3.1 states the same contract
for its `Hash`. Its cache keys an entry by the hash of the call set
combined with the state's hash. A set listed in insertion order is a
model whose `Equal` is coarser than the standard's. Its `Hash` reads the
parts that its `Equal` reads. Without a `Hash`, its memo runs at the
speed of the slow memo that the definition's RFC-0004 measured.

The search compares two states of one step with `Equal` only when their
hashes are equal. Two states of different hashes are then two states,
whatever `Equal` reports. The search copies the states that `Step`
returns into its store, and keeps no slice that `Step` returns. A model
may return one shared slice for a list of states that it leaves often,
as long as nothing writes to the slice.

`ModelFrom` returns a model whose state is the list of the operations
and arguments applied so far. Each is an `Op` with no output. A step
builds a subject with `factory` and applies the list and then the call.
The step accepts the call when the subject's output equals the recorded
output under `equality.Equal`. It accepts a call whose outcome is
unknown. Two states are equal when they list equal operations with
equal arguments in one order. A step at depth d costs d + 1 steps of the
budget. `Model` keeps that cost in an unexported field, and every other
model costs one step.

`equality.Hash` walks a value as `Equal` does:

- It hashes -0 and +0 alike.
- It hashes a map's entries in any order, a pointer by its target, and a
  function by its code pointer.
- It reads at most 8 levels and 256 parts of a value. The bound ends the
  walk on a value that contains itself.
- A map of more entries than the parts left hashes by its length alone.
  A map lists its entries in no fixed order, so a bounded walk cannot
  pick the same entries of two equal maps.

Two values that `Equal` reports equal under no relaxation have one hash:
they agree on every part that the bounded walk reads.

### The checker

`Linearizable` reads the events and the identities of their keys once,
under the history's mutex. It decides the history as the reference does:

1. **Calls.** It pairs each invocation with its completion. It removes
   each call that failed. An `ok` call is known, with its output. An
   `unknown` or pending call is not known, and is open to the end of the
   history.
2. **Partitions.** It joins two calls that share a key's text, with a
   union-find over the calls in event order. A call that declares no key
   joins every call. Partitions are ordered by their first invocation,
   and each lists its keys in the order the history first declares them.
3. **Search.** It searches each partition with the definition's
   procedure: the entries in a doubly linked list, the candidates in
   event order, the restart from the first entry after each accepted
   call, and the frontier, the first configuration in search order with
   the most linearized calls.
4. **Memo.** A configuration is a bit set of linearized calls and a list
   of states without two equal states. Its key mixes each word of the
   bit set into the wrapping sum of the states' hashes, so the order of
   the states does not change it. The memo maps a key to the newest
   configuration of that key, and each configuration links to the
   previous configuration of its key. Two configurations are one when
   their bit sets are equal and each state of one equals a state of the
   other in a list of the same length.
5. **Limits.** It checks the budget before each step and takes no step
   that passes it. It checks the memo limit before it stores a
   configuration, counting one bit per call of the partition for each.
   It reads the platform clock before the first step of each partition's
   search and every 1,024 steps after it, and ends the search as
   undecided with `LimitTime` once the time limit has passed.
6. **Workers.** With `Workers(n)`, up to n goroutines search the
   partitions in partition order. The checker reads their results in
   partition order and reports what one worker reports: the first
   violated partition, else the first undecided one, with the steps of
   the partitions up to and including it. Once it has read the result
   that it reports, or a fault, it cancels the searches still running.
   A search reads the cancellation with the clock and stops. The checker
   takes the partitions that no worker has taken, and a search that a
   worker starts after the cancellation stops before its first step.
7. **Faults.** It recovers a panic in `Init`, `Step`, `Equal`, `Hash` or
   a subject of `ModelFrom`, and a function of the model that ends its
   goroutine with `runtime.Goexit`. It ends the call with a fault of the
   kind `ErrModel` that names the function and the call that the search
   stepped, with its operation:
   `history.Linearizable: calls[4]: the model's Step panics on "read" with runtime error: index out of range [0] with length 0`.
   A panic in a partition that one worker would not have searched does
   not end the call.

| Condition | Behaviour |
|---|---|
| A partition's search uses up the budget | `Undecided`, with `LimitSteps` |
| A partition's memo would pass the memo limit | `Undecided`, with `LimitMemo` |
| The check passes the time limit | `Undecided`, with `LimitTime` |
| A panic in a model's function or a subject, or a function that ends its goroutine | A fault of the kind `ErrModel`. The call record states the verdict `error` |
| A nil history, or a model without `Init` or `Step` | A fault without a kind |
| No calls, or only calls that failed | A pass |

With n workers, `Step`, `Equal` and `Hash` run on up to n goroutines at
once. A model is then safe for concurrent use, and a model of pure
functions is. On one worker, the search runs on the caller's goroutine,
so a function of the model that calls `t.FailNow` ends the test there.

### The memory of a search

A search keeps its states and its configurations in four arrays and a
map. Its allocations are the growth of these and the slices that `Step`
returns.

| Structure | Contains | Size |
|---|---|---|
| Store | The states of every configuration, each configuration's states in one run, the initial state first | One `S` per state |
| Configurations | The first word of a configuration's bit set, its run of states in the store, and the index of the previous configuration of its key | 32 bytes per configuration |
| Words | The further words of each bit set, one for every 64 calls past the first 64, at the configuration's index times their number | 8 bytes per further word |
| Memo | A Go map from each key to the index of the newest configuration of the key | 16 bytes per key, with the map's control bytes and free slots |
| Buffer | The states that the current step leaves, with their hashes | Every step reuses it |

A step leaves its states in the buffer. The search copies them to the
store only for a configuration that the memo lacks. A frame of the stack
and the frontier each refer to a run of the store, so a backtrack
restores the states without a copy. The record copies the frontier's
states once. Every configuration that the search stores follows at least
one step, so a search of the default budget stores at most 10,000,000
configurations in 305 MiB of records.

### The record

`Linearizable` aborts only, as `prop.ForAll` does. A failing check
reports one record of the assertion `linearizable`, with the caller's
contract and the ten detail fields of the definition:

| Field | Go type |
|---|---|
| `outcome` | `Verdict`: `Violated` or `Undecided` |
| `partitions`, `steps`, `calls`, `concurrency` | `int` |
| `partition` | `[]any`, the keys of the reported partition. Empty means every key |
| `linearized`, `candidates` | `[]Span` |
| `states` | `[]S` |
| `limit` | `Limit` for an undecided check, and nil for a violated one |

`Verdict` and `Limit` are `uint8` enumerations with a `String` method
from stringer, whose line comments are the definition's spellings.
`Span.Completion` is -1 for a pending call, and a call whose outcome is
unknown has an `Op` that is not known.

The call record of a failing check states the detail in the history's
JSON form, which `Span.MarshalJSON` and a marshaler of the whole detail
write, as `prop.ForAll` states a run through `matcher.Running.FailRun`.
A record in a recorded run then states what the vector of the same
history states.

`history` registers the sentence of `linearizable` with
`matcher.RegisterSentence` in its `init` function, as `prop` registers
its sentences. The text of a frontier is free under the definition, and
reads:

```text
the register is linearizable: violated in the partition of "x"
    steps 2, partitions 1, calls 2, concurrency 1
    linearized: call 0 write(1) → <nil>
    states: 1
    rejected: call 2 read() → 0
```

### Conformance

`conformance.Vectors` reads `spec/corpus/history/*.json` beside the
property engine's files, and `VectorKind` gains `seam` and
`linearizable`:

| Kind | What the runner does |
|---|---|
| `seam` | Records the script through `New`, `Invoke`, `OK`, `Fail` and `Unknown`, or builds the history with `FromIntervals`. It compares each event's JSON form with the vector's events. For a refused script it recovers the panic and compares the entry. For refused intervals it compares the index in the fault's path |
| `linearizable` | Records the history's script, builds the named model, and runs `Linearizable` with an `assert.Recorder` and the vector's options. A failing vector compares each field of the detail of the call record with the vector's. A passing vector requires a call record of a pass, and checks the steps through the budget |

A passing check reports no record, so the runner observes a pass's steps
through two more checks of the same history. Under `Budget(steps)` it
passes, and under `Budget(steps - 1)` it ends as undecided after
`steps - 1` steps, with the vector's number of partitions. The budget
applies to each partition, so this fixes the steps of a pass over one
partition. Every passing vector of definition 3.1.0 has one partition. A
passing vector over more than one partition is a fault of the runner,
which states that no record of a pass states its steps.

`conformance.Models` builds the six named models natively, as the
`models` section of the definition states them. Each is a `Model[any]`,
and two values compare by their canonical texts:

| Model | State |
|---|---|
| `register`, `cas-register`, `lossy-register` | The value |
| `key-value` | `literal.Pairs`, which keeps its keys in the order they were stored, with an `Equal` and a `Hash` that ignore the order |
| `queue` | `[]any`, head first |
| `set` | `[]any` in the order of adding, with an `Equal` and a `Hash` that ignore the order |

### Testing

- Every source file has a black-box test file beside it, in the package
  `history_test`.
- The vectors pin the recording order, the processes, `FromIntervals`,
  the search, its steps, its limits and its record. A unit test pins
  each contract that no vector states: a reading without a gap while
  eight clients record, the panic of each usage error, a key without a
  typed literal, the driver's deadline, its outcomes and the panic it
  raises again, a body that calls `runtime.Goexit`, the time limit, the
  memo's comparison of the states of one set of calls, a set of more
  than 64 calls, the defaults of every predeclared type with -0, +0 and
  NaN, and the fault of each function of a model.
- A test of the workers forces a later partition to finish first: the
  step of partition 0 waits on a channel that the last step of partition
  1 closes, so the checker reads partition 1's result first on every run
  and still reports what one worker reports. A test of the cancellation
  makes the first step of partition 0 wait for the first step of
  partition 1, and then requires partition 1's search to stop long
  before its budget.
- The release has no effect that a test observes on every run, because
  the operating system's scheduling sets how close the bodies' first
  calls are. gremlins makes no mutant of a channel's close or receive,
  and no test claims to pin the release.
- The tests of the driver wait on channels, never on `time.Sleep`. A
  body that runs past the deadline blocks on a channel that the test
  closes after `Concurrently` returns. The tests of the time limit wait
  on a timer of the platform clock inside a model's step, because the
  limit reads that clock.
- Every exported function and method has a benchmark under
  `bench.Start(b).MaxAllocs(n)`. Its doc comment states the
  allocations. A passing check of a register over two sequential calls
  allocates 38 times.
- A benchmark spends the default budget on the first history of the
  definition's RFC-0004 measurements. Eighteen clients write distinct
  values at once, and a read then outputs a value that no write wrote.
  The check takes 0.53 to 0.72 seconds in 12 runs on one processor, with
  a median of 0.54 seconds, or 54 ns per step. Porcupine takes 0.60 to
  0.70 seconds in RFC-0004. The check allocates 10.0 million times and
  437.7 MiB. Almost every allocation is the slice of one state that the
  register's `Step` returns. The second history of RFC-0004 is a
  partition of Porcupine's `kv/c50-bad`. Its benchmark needs a reader of
  Porcupine's log format, and this repository has none.
- A second benchmark checks a queue whose states are `[]int`, which the
  defaults compare and hash through reflection. Ten clients enqueue
  distinct values at once, and a dequeue then outputs a value that no
  client enqueued. The check spends a budget of 1,000,000 steps in 0.16
  to 0.21 seconds in 12 runs, with a median of 0.17 seconds. It
  allocates 1.47 million times and 361.5 MiB.
- The coverage stage requires 100% statement coverage of `history`.
  gremlins measures `history` on demand against the bar of 100%, as it
  measures the property engine. The mutation stage of `ergon check`
  covers `internal/equality` and `conformance`, the packages of the hash
  and of the runners.

### Changes to the standards

`docs/standards/code.md`, Packages: the public packages are `assert`,
`expect`, `golden`, `bench`, `prop` and `history`.

`docs/standards/errors.md`, Panics, gains two rules before "No other
code panics on purpose":

> The history panics when a caller breaks its contract: an invocation by
> a client whose call is open, a second completion of one call, a key
> that no typed literal states, and `Concurrently` with fewer than one
> client or a negative time. Its message starts with the package and
> names the call. `Concurrently` raises a body's panic again on the
> caller's goroutine.
>
> A named model of `conformance` panics on an operation that it does not
> define, and the check reports the panic as a fault of its model.

The rule on `recover` lists three more places where a panic is what the
code observes: a body of `Concurrently`, a function of the model in
`Linearizable`, and a call of the history that the seam runner of
`conformance` makes to read the entry that a script refuses. The
Writing section states that `history` registers the sentence of
`linearizable` as `prop` registers its sentences.

### The overlay

The overlay of definition 3.1.0 states one limit of the history: the
mutex synchronizes the clients, and that can supply a memory barrier
that the subject lacks. It declines no name of the history, the driver
or the checker.

## Alternatives considered

### A. The search in an internal package

The calls, the partitions and the search are in `internal/history`.
`conformance` calls the search directly and compares the steps of every
pass.

**Why not:** the search reads events, and an internal package cannot
name the public `Event`. The internal package would declare an event, a
kind and an op of its own. The public package would copy each event into
that form on every check. The copy would make one more value observable:
the steps of a pass over more than one partition. No record states that
value, and no vector of definition 3.1.0 has such a pass.

### B. A model as an interface

`type Model[S any] interface { Init() S; Step(S, Op) []S }`.

**Why not:** a test then declares a named type for each model, where a
struct literal states the model inline at the call. A struct also makes
`Equal` and `Hash` optional with a nil field. Porcupine's models are
structs of functions for the same reasons.

### C. Comparable states, hashed by Go's maps

`Model[S comparable]`, with the memo keyed by the state itself.

**Why not:** a queue's state is a slice and a key-value state is a map,
and neither is comparable. Go's `==` also compares a pointer by its
address, where the definition compares its target.

### D. A lock-free recorder

An atomic counter assigns the index, and each client writes its event
into its slot and marks the slot written. `Events` returns the events up
to the first unwritten slot.

**Why not:** the mutex gives the same order and a reading without a gap
in a few lines. Both synchronize the clients alike, which the overlay
declares, and no measurement shows the mutex's cost in a test that
records calls.

### E. A client's panic wrapped with its stack

The driver raises a value that contains the client's number, the
panic's value and the goroutine's stack.

**Why not:** the definition raises the panic again, and a `recover` in
the caller then sees a value that the body did not raise.

### F. An end that a pending interval omits

`End *int64`, nil for a pending entry.

**Why not:** a pointer makes every completed entry take the address of a
time. The kind already states whether a call completed.

### G. A `Hash` required beside `Equal`

A model that states `Equal` without `Hash` ends the check with a fault.

**Why not:** a test of a few dozen calls checks in a few hundred steps
with the slow memo, and a fault would make every small model state a
hash. Porcupine also makes its `Hash` optional.

### H. A public check that returns its verdict

`history.Check(h, m, opts...) Verdict` returns the steps and the
frontier without a test.

**Why not:** no caller needs a verdict outside a test, and the only
caller would be `conformance`. The budget states what a pass's steps
are.

### I. A memo entry per configuration

The memo maps a key to a slice of entries, as Porcupine's cache does.
Each entry contains a copy of the bit set and the slice of states that
the step built.

**Why not:** each stored configuration then allocates a copy of its bit
set and a slice of its states. Each new key allocates a slice of
entries, and the collector scans every entry for the pointers in it.
With this memo, and every state compared and hashed through reflection,
the register benchmark takes 1.34 to 1.42 seconds and allocates 41.3
million times and 589.8 MiB.

### J. Arrays that double

The store, the configurations and the words double their capacity when
they grow, where `append` adds about a quarter.

**Why not:** doubling allocates about twice an array's final size,
against five times. While a copy runs, an array and its successor take
three times its size, against 2.25 times, so the peak of a search
rises. The capacity check also adds a boundary mutant that changes only
a capacity, which no black-box test observes.

## Drawbacks

- **A state of any other type compares and hashes through `reflect`.**
  A named type over `int` is such a type. On the register benchmark,
  `int` states take 53 to 57 ns per step in six runs, and the same
  values of a named type take 74 to 78 ns. The doc comment of `Model`
  states that such a model is faster with an `Equal` and a `Hash` of its
  own.
- **A model with its own `Equal` and no `Hash` searches as slowly as
  the definition's slow memo.** The doc comment of `Model` states it.
- **The arrays of a search grow by `append`.** Go 1.27 grows a slice of
  more than 256 elements by about a quarter of its capacity, so an array
  allocates about five times its final size over a search. Each array
  that it outgrows is garbage. In a profile of the register benchmark,
  the configurations account for 55% of the bytes that the check
  allocates. The queue benchmark allocates 361.5 MiB, while its
  collector finds at most 140 MiB live and its process peaks at 215 to
  236 MiB.
- **The driver loses a panicking body's stack.** The panic raised again
  shows the driver's stack.
- **Conformance checks the steps of a pass over one partition only.** A
  vector of a pass over more than one partition makes the runner fail
  until the design changes.
- **The mutex serializes recording.** Clients that record at once wait
  for each other, which is the synchronization that the overlay
  declares.
- **`Invoke` encodes each key.** A key of a string or an int costs the
  literal's allocations on every invocation.

## Open questions

None.

## Unresolved and future work

- Machines, simulation and campaigns, the definition's RFC-0012, whose
  concurrent sections record through this history, are not proposed
  here.
- The isolation checks over transaction histories, the definition's
  RFC-0015, are not proposed here.

## References

| What | Where |
|---|---|
| The observation seams | The definition's RFC-0003 |
| The linearizability checker, its search, limits, record and measurements | The definition's RFC-0004 |
| The executable reference and the 42 vectors, version 3.1.0 | <https://github.com/dokimasia/assert-spec>, `tools/history` and `corpus/history` |
| The property engine in Go, whose structure this RFC follows | RFC-0002 |
| One equality for every comparison | RFC-0008 |
| Porcupine v1.3.1, `Model.Hash` and the cache key | <https://github.com/anishathalye/porcupine/blob/v1.3.1/model.go>, lines 121 to 124, and `checker.go`, lines 214 to 220 |
| MurmurHash3's final mix, `fmix64` | <https://github.com/aappleby/smhasher/blob/master/src/MurmurHash3.cpp>, lines 86 to 94 |
| `hash/maphash` | <https://pkg.go.dev/hash/maphash> |
| The growth of a slice in Go 1.27.1 | `runtime/slice.go`, `nextslicecap`, lines 326 to 358 |
| `runtime.Goexit` | <https://pkg.go.dev/runtime#Goexit> |
| `sync.Mutex` and the Go memory model | <https://go.dev/ref/mem> |
