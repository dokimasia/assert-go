---
rfc: 0011
title: Machines, the task scheduler and campaigns in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-05
updated: 2026-10-05
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0011: Machines, the task scheduler and campaigns in Go

## Summary

Version 3.3.0 of the definition adds machines, the task scheduler,
traces of steps and campaigns to the property engine. This RFC implements
them in Go:

- **`go.dokimi.dev/assert/stateful`, a new public package**, contains
  `Machine`, `Action`, `Steps` and its seven options, and the task
  scheduler with its two strategies. The naming table names them in
  `stateful`.
- **`prop` gains the case's `History` and `Target`**, step entries in the
  text of `Draws`, the steps of a counterexample, and the `campaign`
  profile.
- **`history` gains two options of `Linearizable`**: `Whole`, which
  searches the history as one partition, and `Final`, which returns the
  states that the order found leaves. A machine's check after each step is
  a run of `Linearizable` with both.
- **`conformance` runs the 13 machines vectors** with the six machine
  subjects built natively.

The definition fixes the six parts of a run of steps, every choice they
make, the scheduler's strategies, how a trace's step entries turn back
into choices, and what a counterexample lists. This library translates
the executable reference in `tools/stateful` into Go. The module gains no
dependency.

The design makes five decisions that the definition leaves to Go:

- **A machine is generic over its model's state.** `Machine[S]` takes a
  `history.Model[S]`. The zero model checks nothing.
- **A task is a goroutine that runs only while the scheduler releases
  it.** One goroutine runs at a time, so a scheduled run is as
  deterministic as a run on one thread.
- **A section on real threads starts its clients with
  `history.Concurrently`**, and the engine runs such a case up to `Repeat`
  times.
- **The check after a step reports the record of `linearizable`** to the
  case through a seat of the machine's own. A passing check writes no call
  record.
- **A counterexample is a list of entries.** `prop.Entry` is a `Drawn`
  value or a `Step`, in request order.

The RFC amends the list of public packages in `docs/standards/code.md`,
and the rules on panics and on `recover` in `docs/standards/errors.md`.

## Motivation

The definition states what a run of steps does. Go has to settle how:

- **How a machine reads in a Go test.** A model's state can be of any
  type, and an action's input is whatever the action requests.
- **How a task yields.** Python and TypeScript suspend a coroutine at an
  `await`. A goroutine has no such point, and the runtime schedules every
  goroutine itself.
- **How a machine makes the case's choices.** The engine's requests are
  internal, and `stateful` is a public package beside `prop`.
- **How the check after a step returns the states.** The search is
  unexported code of `history`. A check that passes reports no record,
  and a machine needs the states that the order found leaves.
- **How a section on real threads repeats a case.** The definition runs
  such a case `repeat` times, and the engine runs each case once.
- **How a counterexample lists steps.** `prop` states a counterexample as
  `[]Drawn`, and a machine's counterexample lists its steps among its
  draws.
- **The names that the naming table does not fix**, such as the type of a
  strategy and the type of a step in a counterexample.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| Machine | `stateful`, new | `Machine`, `Action`, `Steps` and its seven options, and the six parts of a run of steps |
| Task scheduler | `stateful` | `Scheduler`, `NewScheduler`, `Spawn`, `Yield`, `Run`, and the strategies `Uniform` and `PCT` |
| Case members | `prop` | `Case.History` and `Case.Target` |
| Machine requests | `internal/prop/engine` | The swarm choice, the index by weight, the continue flag of a run of steps, a uniform choice, a span from an earlier choice, the record of a step, the repetition of a case, and the trace of the case of `Draws` |
| Traces | `prop`, `internal/prop/engine` | Step entries in the text of `Draws`, their refusals, and the steps of a counterexample |
| The check after a step | `history` | The options `Whole` and `Final` of `Linearizable` |
| Campaigns | `internal/prop/engine`, `prop` | The campaign with its pool, its mutations and its failures, and the `campaign` profile with its budget |
| Vectors | `conformance` | The machines vectors, the six machine subjects, and the overlay's `sections` |

`stateful` imports the root package for `TB`, `prop`, `history`, the
internal packages `engine`, `choice` and `random`, and the standard
library. `engine` and `prop` import `history` for the case's history. No
package of the module imports `stateful` but `conformance` and the tests.

### The public surface

```go
package stateful

// Machine is a model of a subject and the actions over the subject.
type Machine[S any] struct {
	Model     history.Model[S] // a Model without Init and Step checks nothing
	Actions   []Action[S]      // in order, the simpler first
	Invariant func(c *prop.Case, state S)
	Settle    func(c *prop.Case, state S)
}

// Action is one named action of a machine.
type Action[S any] struct {
	Name    string
	Weight  int // the relative frequency, 1 when 0
	Enabled func(state S) bool
	Drain   bool
	Input   func(c *prop.Case, state S) any
	Run     func(c *prop.Case, client int, input any)
}

func Steps[S any](c *prop.Case, m Machine[S], opts ...Option)

// Option configures one run of Steps.
type Option struct{ /* unexported */ }

func Mean(n int) Option
func Max(n int) Option
func Swarm(on bool) Option
func Clients(n int) Option
func Concurrent(n int) Option
func Tasks(s *Scheduler) Option
func Repeat(n int) Option

// Scheduler releases the tasks of one case, one at a time, in an order
// that the case's choices decide.
type Scheduler struct{ /* unexported */ }

func NewScheduler(c *prop.Case, strategy Strategy) *Scheduler
func (s *Scheduler) Spawn(task func())
func (s *Scheduler) Yield()
func (s *Scheduler) Run()

// Strategy is how a scheduler chooses the next task to release. The zero
// Strategy is Uniform.
type Strategy struct{ /* unexported */ }

func Uniform() Strategy
func PCT(depth int) Strategy
```

```go
package prop

func (c *Case) History() *history.History
func (c *Case) Target(label string, score float64)

// Entry is one entry of a counterexample: a Drawn value or a Step, its
// only two types.
type Entry interface{ /* unexported methods */ }

// Step is one step of a machine in a counterexample.
type Step struct {
	Action string
	Client int // the client of a concurrent step, and -1 for any other step
	Drain  bool
}
```

The field `Counterexample` of `Other`, and the `counterexample` field of a
run's detail, become `[]Entry`.

```go
package history

func Whole() Option
func Final[S any](states *[]S) Option
```

The naming table fixes `Machine`, `Action` and their members, `Steps` and
its seven options, `Scheduler` with `NewScheduler`, `Spawn`, `Yield` and
`Run`, `Uniform`, `PCT`, and the case's `History` and `Target`. This RFC
decides the other names:

| Name | Why Go needs it |
|---|---|
| `stateful.Option` | The seven options are values of one type, as `prop.Option` is |
| `stateful.Strategy` | `NewScheduler` takes the value that `Uniform` and `PCT` return |
| `prop.Entry`, `prop.Step` | A counterexample lists steps among its draws, and a step is a value of its own |
| `history.Whole`, `history.Final` | A machine's check searches the history as one partition and reads the states that the order found leaves |

### A run of steps

`Steps` runs the six parts of the definition on the goroutine that calls
it, the body's, and makes every choice there:

1. **Swarm.** One choice per action, as the definition's `keep` draws it,
   with probability 1/2, edge 1 and target 0. A trace keeps the actions
   that its step entries name.
2. **Setup.** The check and the invariant.
3. **Sequential steps.** Before each step, the actions that swarm kept and
   that `Enabled` accepts in every state, each `Enabled` called once per
   state. With none, the steps end and make no choice. Otherwise a
   continue flag by the collection rule, with minimum 0, maximum `Max` and
   average `Mean`, then an index by weight with edge 0. The step is a span
   labelled with the action's name, from its flag to the end of its run.
   `Input` requests the step's input from the first state, and `Run` calls
   the subject on client 0. The check and the invariant follow.
4. **The concurrent section**, with `Clients` of 2 or more: every step
   listed first, each a flag with maximum `Concurrent` and the collection
   rule's average, a client uniform over 0 to `Clients` with edge 0, and an
   index over the kept actions without `Enabled`, with its input. Client 0
   runs its steps on the body's goroutine, and then the other clients run
   theirs at once. The check follows.
5. **The drain.** The actions with `Drain` set, kept or not, take steps
   without a flag until none is enabled or `Max` drain steps have run.
6. **Settling.** `Settle` runs on the first state of the last check, then
   the invariant.

`Weight` of 0 counts as 1, so an action literal that states no weight has
the definition's default. A machine whose model states neither `Init` nor
`Step` checks no history. Its functions receive the zero `S`. A model
that states only one of them ends the case with the fault of
`Linearizable`.

`Steps` panics for a machine that states no machine: an action with a
negative weight or with a nil `Run`, and two actions with one name. The
panic fails the case, so the property fails at its first case with the
panic's message, such as
`stateful: Steps of a machine whose actions name "put" twice`. Each
option panics for a count below its minimum: `Mean`, `Max` and
`Concurrent` below 0, and `Clients` and `Repeat` below 1, as
`prop.Workers` panics below 1. `Tasks` panics for a nil scheduler.

### The concurrent section

**On real threads**, without `Tasks`, the clients 1 to `Clients` that
have a step run through `history.Concurrently`, one goroutine each,
released together. The section states no time limit, so `Steps` passes
the largest `time.Duration`, and the test's own deadline applies. A
client that panics makes `Concurrently` raise the panic again on the
body's goroutine, which fails the case. A client that ends its goroutine,
as a fatal record of the case does, has finished, and the case keeps its
record.

For a case whose section starts a thread, `Steps` calls the engine's
`Repeat` with the count of the option `Repeat`, 4 by default. A section
whose every step is on client 0 starts none, and its case runs once.
When a case that called `Repeat` passes, the engine runs the body again on
the case's choices, outside the case tree, until a run fails or `Repeat`
runs have passed. The case is then the first run that failed, with its
record and its calls. The case keeps the largest count of its calls. The
replay before shrinking and every shrink candidate repeat the same way,
because each run of the body calls `Repeat` again. A case that failed once
and then passes every repeat of its replay ends the run as `flaky`.

A run on more workers runs a case ahead of the runner, outside the case
tree, and enters its walk into the tree once the case has ended. Its
repeat keeps its walk too, and states in each call record the steps of
that walk, so a repeat that the tree stops at a tested case keeps the calls
that a run on one worker makes.

**As tasks**, with `Tasks(s)`, `Steps` spawns one task per client that
has a step, in client order, and calls `s.Run()`.

### The task scheduler

A task is a goroutine that the scheduler starts at `Spawn`. The goroutine
waits for its release, and then runs until it calls `Yield` or ends. The
scheduler releases one task at a time and waits for it. At any moment one
goroutine of the case runs: the body's inside `Run`, or the released
task's. The goroutine that runs `Run` makes the choice of each release,
and the goroutine that calls `Spawn` makes the priority of `PCT`. The
choices follow the order of the releases, which a replay repeats.

| Call | What it does |
|---|---|
| `NewScheduler(c, strategy)` | A scheduler that releases tasks by strategy, with the choices of the case c |
| `Spawn(task)` | Starts task's goroutine and makes it ready, after every ready task. Under `PCT`, the task first receives its priority, an integer choice over [0, 2^64 − 1] with target 0 |
| `Yield()` | In the running task, makes it ready again, after every ready task, and waits for its next release. Outside a task, on the body's goroutine, returns at once, so a step on client 0 runs to its end |
| `Run()` | Makes `PCT`'s depth − 1 change points, then releases one ready task at a time until none is ready |

`Uniform()` chooses each release uniformly among the ready tasks, with
target 0 and edge 0. A release with one ready task records a choice that
consumes nothing. `PCT(depth)` releases the ready task with the highest
priority, the earliest ready among equals, and makes no choice at a
release. Each change point is a presence choice with edge 1 and then a
count of releases, and `Run` makes them even when no task is ready. The
task released at that count falls below every other task, and a later
change point puts its task lower still. `PCT` panics for a depth below 1.

When a task returns, ends its goroutine or panics, `Run` acts as follows:

| A task | `Run` |
|---|---|
| Returns | Releases the next ready task |
| Ends its goroutine, as a fatal record, a rejection or a draw that ends the case does | Ends the goroutine that runs `Run` the same way, after it ends every other task |
| Panics | Recovers the panic on the task's goroutine, ends every other task, and raises the panic again on the goroutine that runs `Run` |

`Run` panics when a task calls it on its own scheduler. A task that is
spawned and never released ends when its case ends. The scheduler
registers a cleanup with the case, which ends every task still waiting,
so every goroutine of a case has ended once the case has. A task that the
scheduler ends inside `Yield` ends its goroutine there, so its deferred
calls run. A task that it ends before its first release runs nothing.

A subject that runs as tasks takes the scheduler or an interface with
`Spawn` and `Yield`. It calls `Spawn` where it would use the `go`
statement, and `Yield` at each point where another task may run. The
racy counter of the definition reads the count, yields, and writes the
count plus one:

```go
increment := func(c *prop.Case, client int, _ any) {
	call := c.History().Invoke(client, "increment", nil)
	n := count
	s.Yield()
	count = n + 1
	call.OK(count)
}
```

### The check after a step

A machine with a model checks the case's history after setup, after
every sequential and drain step, and after the concurrent section:

```go
history.Linearizable(&seat, c.History(), m.Model, contract,
	history.Whole(), history.Final(&states))
```

- `Whole()` searches the whole history as one partition, whatever keys
  its calls declare. The model is one state of the whole subject.
- `Final(&states)` stores the states that the first order found leaves,
  for a check that passes and searched one partition. A history without
  calls leaves the initial state.
- The seat is the machine's own. It embeds the case as an `assert.TB`,
  which receives every report of the check but its record, such as the
  fault of a model that panics. It keeps the record, and writes no call
  record, so the many passing checks of a run leave nothing in a recorded
  run.
- A check that fails, violated or undecided, ends the step: `Steps` passes
  the record of `linearizable` to the case with `Report`, aborting. The
  case's failure is then the record of `linearizable`, whose identity is
  the assertion and the call site of `Steps`.

The contract of the check is `the history of the machine's steps is
linearizable`. The check runs with the defaults of `Linearizable`: a
budget of 10,000,000 steps and a memo limit of 2^33 bits.

### The case's history and target

`Case.History` returns the case's history, which is empty when the case
starts. The engine creates it at the first call, and drops it when the
run reuses the case's storage for a later case. `Case.Target` records a
score of the case under a label, and the case keeps the higher of two
scores of one label. A campaign reads the scores. An ordinary run records
them and generates as if they were absent.

### Traces

`Draws` reads step entries beside draw entries, in the definition's
form:

```go
prop.Draws(`[
	{"label": "capacity", "value": {"type": "int", "value": 2}},
	{"step": "put"},
	{"label": "v", "value": {"type": "int", "value": 0}},
	{"step": "put", "client": 1},
	{"step": "deliver", "drain": true}
]`)
```

An entry that states `step` is a step entry, and every other entry is a
draw entry. A step entry's client is 0 or more, because `Client` of -1
states a step outside a section in Go. `Draws` refuses a negative client
with a fault at the entry's `client`, before the run.

The engine's entry of `Settings.Draws` gains a step form, and the case of
`Draws` follows its entries through a trace. `Steps` reads the next entry
before each step. For a step entry of its part, it takes the entry and
serves the step's flag, its client, and the index of the action that the
entry states to the step's requests. A step entry that the run of steps
cannot take ends the case with a refusal: a fault at the entry's `step`,
which ends the run before any other case:

```text
prop.ForAll: Draws[1].step: "flush" is not among the actions that the step lists
```

| The step entry | Reason |
|---|---|
| Is a sequential step at the maximum | `the sequential steps are at their maximum of 100` |
| Names an action that the step's list lacks | `"flush" is not among the actions that the step lists` |
| Is a concurrent step at the maximum | `the concurrent section is at its maximum of 16` |
| States a client outside the clients | `client 3 is outside clients 0 to 2` |
| Is a concurrent step of a machine with one client | `the machine runs no concurrent section` |
| Is a drain step after the drain | `"flush" is not among the drain actions that are enabled` |

A draw that takes a step entry refuses it at the entry's `label`, with the
reason `the draw labelled "count" takes the step entry of "put"`.

### The steps of a counterexample

The engine records each step with the number of draws before it, and
`prop` lists the counterexample in request order: a step before the draws
that the case recorded after it. A `Step` states its action, its client
in a concurrent section and -1 elsewhere, and the drain mark. Its JSON
form is the definition's: `{"step": "put"}`, with `"client"` in a
concurrent section and `"drain": true` in the drain. The explain phase
explains the draws, and a step states no relevance. The sentence of a
run's record writes a line for each step, such as `step put on client 1`.

A store entry lists the draws of its counterexample, as the definition's
store format states, and no step. Its choices replay the steps.

### Campaigns

`DOKIMI_ASSERT_PROP_PROFILE=campaign` makes every run of `ForAll` and of
a property form a campaign. `DOKIMI_ASSERT_PROP_BUDGET` states how long
each campaign runs, in whole seconds, as a decimal number from 1 to
2^33 − 1. A campaign without a budget, or with one outside those, fails
the test before any case runs with a fault at the variable. The profile
draws each seed as the default profile does. `Fuzz` runs no campaign and
reads no budget.

`engine.Campaign` runs the case of `Draws`, the examples and the stored
cases in the order of a run, and then explores until `Settings.Budget`
has passed on `Settings.BudgetClock`. `prop` leaves that clock at its
default, the platform clock, and the engine's tests set a controlled
clock.

- **The pool.** A valid case joins the pool when it counts a label that no
  earlier case counted, records a fingerprint that no earlier case
  recorded, or reports a score above the best for its label.
- **The cases.** Once the pool has a member, three cases in four mutate a
  member, chosen uniformly. The fourth is the next random case of the
  run's seed: random case i for i = 0, 1, 2 and on, past `Cases`. Random
  case i of a seed is the stream of the seed plus i, so moving to the next
  seed would repeat all but one of the seed's cases.
- **The mutations.** One choice drawn again from its bounds, a span
  deleted, a span repeated, or a span replaced by a span with the same
  label of a member, each with probability 1/4. A member without spans,
  and a replacement whose donor has no span of the label, draws a choice
  again. A member without choices takes no mutation, and every request of
  its case draws from the campaign's stream. A mutated case replays its
  choices, each fitted to its request's bounds, and each request past
  them draws from the campaign's stream, the stream of its seed's case
  2^64 − 1.
- **The failures.** A failing case whose identity no earlier failure had
  is replayed, shrunk and explained, as a run concludes its failure, and
  the campaign goes on. `prop` stores it as the campaign concludes it,
  through the engine's `Settings.Concluded`. The other failures that its
  shrink finds join its others, except those of an identity concluded
  before. A replay that differs ends the campaign as `flaky`.

When the budget has passed, the campaign reports one record of its
property: `counterexample` with the first failure and every later one
under `others`, or the outcome that a run's last check decides over every
valid case of the campaign. The record states the campaign's valid and
rejected cases, and its seed. The mutations and their probabilities are
this library's, which the definition leaves free, so a campaign's search
reproduces in no other language. Every failure it stores replays in every
language.

A campaign runs one case at a time, outside the case tree, so it can run
one case twice. `Workers` sets the workers of each failure's shrink.

### Machine requests in the engine

`stateful` converts a `*prop.Case` to the engine's case, as `prop`
converts its own, and makes its requests through these members of
`engine.Case`:

| Member | Request |
|---|---|
| `Keep(kept bool, remaining int) bool` | The swarm choice: edge 1, bounds [1, 1] for the last action while no earlier one is kept, drawn as `keep` draws |
| `Weighted(weights []uint64) int` | An index by weight, with edge 0, drawn as `weighted` draws |
| `Continue(count, maxSteps, mean int) bool` | The continue flag of a run of steps, by the collection rule with minimum 0 and the average mean |
| `Uniform(n, edge uint64) uint64` | A choice below n that decides structure, drawn as `below` draws |
| `Integer` and `Structure` | The priority of a task and the count of a change point as integer choices, and the presence of a change point as a choice that decides structure |
| `Position() int` and `SpanFrom(start int, label string, f func())` | A span that starts at an earlier choice, such as a step's flag |
| `Step(s MachineStep)` and `Steps() []RecordedStep` | The record of a step after the draws so far: its action, its client and its drain mark, with the number of draws |
| `Repeat(n int)` | Asks the run to run the case up to n times |
| `Trace() (Trace, bool)` | The trace of the case of `Draws`: `Names`, `Next`, `Prepare`, `Take` and `Refuse` |

The definition's draws `keep`, `weighted`, `below` and the collection flag
with an average become drawings of `engine`'s requests, beside the coin
and the flag that it draws now.

### Conformance

`conformance.VectorKind` gains `Machines`, and `conformance.Vectors`
reads `spec/corpus/stateful/*.json`. The runner of `Machines` builds the
six machine subjects natively, as the `machines` section of the
definition states them:

| Subject | Machine |
|---|---|
| `queue-loses-on-wrap`, `correct-queue` | A ring of a capacity drawn under `capacity`, `put` with an input drawn under `v`, and `get`, over a bounded queue's model |
| `counter-overflows` | `increment` and `reset`, over a counter's model |
| `store-loses-on-crash` | `put`, `flush` as a drain action, and `crash`, with no model and a `Settle` that fails with the record `lost-write` |
| `racy-counter`, `correct-counter` | `increment`, which yields between its read and its write in the racy counter, over a counter's model, as tasks of a scheduler |

Every call of the six subjects completes. The bounded queue's `put`
reports the room that its model counts. The runner's models state those
calls and no other, so the mutation stage finds no condition that no
vector exercises. The failure of `Settle` states one contract for every
lost write, with the key and the lost value in its detail. An assertion's
record without a location takes its contract into its identity. A
contract that contained the key would give each key's failure its own
identity, and the shrink would reject every candidate that loses the
write of another key.

The runner turns the vector's setup into the options of `Steps`, its
settings into the options of `ForAll`, and its trace into the text of
`Draws`. It runs the subject's body as the property of a behaviour
vector, whose store entries are the vector's stored cases, with an
`assert.Recorder` that keeps the fault of a refused trace. It compares the
detail of the run with the vector's, each entry of the counterexample with
a step or a draw. For a refused trace, it compares the fault's path with
the vector's error: `Draws`, the entry's index, and the member that the
vector's reason names. `conformance.Overlay` reads the overlay's
`sections`.

### Testing

- Every source file has a black-box test file beside it, in the package
  `stateful_test`.
- The vectors pin the six parts, the steps a failure shrinks to, the
  scheduler's releases, a trace that fails, a trace whose steps state
  their clients, and a refused trace. A unit test pins each contract that
  no vector states, driven by the choices that it replays: every request
  of each part, its bounds and its edge, each refusal of a step entry,
  `Weight` of 0, each panic, `Yield` outside a task, a task that spawns a
  task, each way a task ends, the cleanup that ends waiting tasks, the
  repetition of a section on real threads, `Whole`, `Final`, and a check
  that fails a step.
- The tests of the scheduler run under the race detector, which the gate
  runs, and wait on channels.
- The campaign's tests run the engine's campaign with a controlled clock
  that each case advances by one second. They pin the random cases of the
  seed in order, and the pool's admission by label, fingerprint and score
  through a search for a list of digits that starts with 7, 3, 9, 1 and 5.
  Guided by any of the three, a campaign finds it within 4,000 to 5,500
  cases of seeds 1, 7 and 42, and without a guide it finds none in 60,000.
  They also pin the mutations of members with spans, without spans and
  without choices, a second failure, a failure that the shrink of another
  finds, a flaky failure in each phase, every outcome, and the faults of
  the budget variable. A test of `prop` runs a campaign of one second under
  the profile and reads its failure from the store.
- Every exported function and method has a benchmark under
  `bench.Start(b).MaxAllocs(n)`, and its doc comment states the
  allocations. One benchmark runs a case of 100 sequential steps of a
  queue's machine with its model, which checks the history 101 times:
  15,272 allocations, about 150 for each check. A case that starts a task
  allocates up to two more when the runtime makes a goroutine instead of
  reusing one, and the scheduler's ceilings include the two.
- The coverage stage requires 100% statement coverage of `stateful`.
  gremlins measures `stateful` on demand against the bar of 100%, and the
  mutation stage of `ergon check` covers the runner of `conformance`.

### Changes to the standards

`docs/standards/code.md`, Packages: the public packages are `assert`,
`expect`, `golden`, `bench`, `prop`, `history` and `stateful`.

`docs/standards/errors.md`, Panics, gains a rule before "No other code
panics on purpose":

> `Steps` panics for a machine that states no machine: an action with a
> negative weight or without `Run`, and two actions with one name. An
> option of `stateful` panics for a count below its minimum, `Tasks` for
> a nil scheduler, and `PCT` for a depth below 1. A scheduler panics when
> a task calls `Run` of its own scheduler. Each message starts with the
> package and names the call.

The rule on `recover` lists one more place where a panic is what the code
observes: a task of the scheduler, whose panic `Run` raises again.

### The overlay

The overlay of definition 3.3.0 states that Go runs a concurrent section
as tasks and on threads, `"sections": ["tasks", "threads"]`, and declines
no name of machines or of the scheduler.

## Alternatives considered

### A. Machines in `prop`

`prop` declares `Steps`, `Machine` and the scheduler beside `ForAll`.

**Why not:** the options of `Steps` would share a namespace with the
options of a run, and need a suffix to tell them apart, such as
`MaxSteps` beside `MaxChoices`. Hypothesis, jqwik and proptest give their
state machines a module of their own. `prop` has 6,209 lines of source
outside its tests already.

### B. A public query of the states

`history.Order(h, m)` returns the states that the first order of the
whole history leaves, and a machine runs `Linearizable` again only when
the query finds no order.

**Why not:** a failing check searches twice, and `history` gains a second
entry point to the search beside the assertion. `Whole` and `Final` are
two options of the one entry point.

### C. The search in an internal package

The search moves to `internal/linearize`, which `history` and `stateful`
both call.

**Why not:** the search reads the public `Op` and the unexported cost
that `ModelFrom` sets in a `Model`. An internal package would declare its
own op and its own model, `history` would copy each model into that form
on every check, and `stateful` could not read the cost. RFC-0009 rejects
the same split for the same types.

### D. A registration hook

`history` registers its search in an internal package at `init`, as
`prop` registers its sentences with `matcher`.

**Why not:** a hook is a function value, and a function value cannot be
generic. Every state would pass as an `any`. The hook would box each
state that a step leaves and compare states through reflection, and a
model of `ModelFrom` would lose its cost.

### E. Tasks scheduled by the Go runtime

A task is a goroutine, and `Yield` calls `runtime.Gosched`.

**Why not:** the runtime chooses which goroutine runs next, so a case
would not replay and would not shrink. The definition takes every release
from the case.

### F. A counterexample step as fields of `Drawn`

`Drawn` gains `Step`, `Client` and `Drain`, and a step is a `Drawn` with
an empty label.

**Why not:** a `Drawn` states a label, a value and a relevance, and a step
has none of them. A caller would test each entry for an empty label
instead of switching on its type.

### G. The check with the case as its seat

`Steps` runs `Linearizable` with the case as the seat.

**Why not:** every passing check would write a call record of
`linearizable` in a recorded run, about 5,000 for a property of 100 cases
of 50 steps.

### H. A section on real threads without the driver

`Steps` starts the clients with its own goroutines and a
`sync.WaitGroup`.

**Why not:** the driver already starts the clients, releases them
together, recovers each panic and raises the lowest client's again. A
second copy would be a second implementation of one concept.

### I. A campaign that stores its failures when it ends

The campaign returns every failure in its result, and `prop` stores them
once the budget has passed, as it stores the failures of a run.

**Why not:** a campaign runs for hours, and a test process that a
deadline or a signal ends before the budget passes would lose every
failure that it found.

## Drawbacks

- **A subject that runs as tasks takes the scheduler.** Its goroutines
  start through `Spawn`, and its yield points call `Yield`. A subject that
  starts goroutines with the `go` statement runs only on real threads.
- **A task costs a goroutine, and each release two channel operations**:
  one that releases the task and one that returns control to `Run`.
- **A section on real threads repeats each case**, up to `Repeat` runs,
  and so does every shrink candidate.
- **The check after a step runs the search on every step.** A case of 100
  sequential steps checks a sequential history 101 times, about 5,000
  model steps.
- **`Steps` panics inside the body for a malformed machine.** The property
  fails at its first case instead of before it.
- **A campaign's search reproduces in no other language**, runs one case
  at a time, and can run one case twice.
- **`prop`'s counterexample changes type**, from `[]Drawn` to `[]Entry`.
  The module has no consumers yet, so the change breaks no caller.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| Machines, simulation and campaigns | The definition's RFC-0012 |
| The executable reference and the 13 machines vectors, version 3.3.0 | <https://github.com/dokimasia/assert-spec>, `tools/stateful` and `corpus/stateful` |
| The property engine in Go | RFC-0002 |
| Property forms and derived inputs in Go, which add `Draws` | RFC-0004 |
| The history seam and the linearizability checker in Go | RFC-0009 |
| The isolation checks in Go, which a machine without a model calls in `Settle` | RFC-0010 |
| Burckhardt, Kothari, Musuvathi and Nagarakatte, PCT, ASPLOS 2010 | <https://doi.org/10.1145/1735970.1736040> |
| `runtime.Goexit` and `runtime.Gosched` | <https://pkg.go.dev/runtime> |
