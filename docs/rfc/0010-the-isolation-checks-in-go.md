---
rfc: 0010
title: The isolation checks in Go
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-05
updated: 2026-10-05
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0010: The isolation checks in Go

## Summary

`go.dokimi.dev/assert/history` implements the two isolation checks that
version 3.2.0 of the definition adds, `serializable` and
`snapshot-isolation`, as `Serializable` and `HasSnapshotIsolation`. Each
reads a history of list-append transactions that the history seam
recorded, derives the dependencies that its reads reveal, and fails with
the first anomaly that its level forbids. The definition fixes the
workload, the derivation, the eleven kinds of anomaly, the cycle search
and the record, and pins them with 38 vectors. This library translates
the executable reference in `tools/history/isolation.py` into Go. The
conformance package runs every vector on every test run. The module keeps
its dependencies: the standard library alone.

The design makes four decisions that the definition leaves to Go:

- **The checks are functions of `history`.** They read a copy of the
  events, taken under the history's mutex, as `Linearizable` does.
- **A value is identified by its typed literal.** Two keys, two appended
  values and two values of a read are one value when `literal.Encode`
  writes one text for them, as the history already identifies keys. A
  check numbers each distinct text once, and the derivation compares
  numbers. A value without a typed literal is a fault of the workload.
- **The graph is an adjacency over transaction numbers.** A transaction is
  numbered by the position of its invocation among the invocations, and
  an edge keeps the first evidence of each of its relations.
- **The record's types are new.** A transaction of the record states its
  completion kind, which a `Span` does not state. The explanation states
  one of two kinds of evidence, an `Edge` of a cycle or an `Observation`.

## Motivation

The definition states what the checks decide. Go has to settle how:

- **How values compare.** The reference compares two values by the
  canonical text of their typed literals, so `int(1)` and `int64(1)` are
  one value and `1` and `1.0` are two. `equality.Equal` would compare the
  first pair as two values, because their types differ.
- **How a fault of the workload ends the call.** The definition refuses a
  call that is no list-append transaction before it derives anything.
- **How the record reads in Go.** The five fields of the detail are Go
  values of declared types, as the fields of every other record are.
- **What a check costs.** The reference spends 29 to 80 µs per
  transaction on histories of 10,000 and 100,000 transactions. A Go test
  of a database records thousands of transactions per run.
- **The names that the naming table does not fix**, such as the
  enumeration of the anomalies and the types of the record's detail.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| Transactions | `history`, unexported | Reads each call as a list-append transaction, numbers the typed literals of its keys and values, and refuses a call that breaks the workload's contract |
| Derivation | `history`, unexported | The committed transactions, each key's version order, the appender of each value, the unobserved appends, and the ww, wr and rw edges with their evidence |
| Direct checks | `history`, unexported | The six anomalies that need no graph |
| Cycle search | `history`, unexported | The strongly connected components, the shortest paths back, and the walk of `G-nonadjacent` with its reduction |
| Record | `history` | The detail of a failing check, its JSON form, and the sentence that the writer renders |
| Vectors | `conformance` | The `serializable` and `snapshot-isolation` vectors |

`history` keeps its imports. `conformance` gains two kinds of vector.

### The public surface

```go
package history

func Serializable(tb assert.TB, h *History, contract string)
func HasSnapshotIsolation(tb assert.TB, h *History, contract string)

var ErrTransaction = errors.New("history: a call that is no list-append transaction")

// Anomaly is a kind of anomaly, in the order in which a record reports
// the kinds.
type Anomaly uint8

const (
	GarbageRead           Anomaly = 1  // garbage-read
	DuplicateAppend       Anomaly = 2  // duplicate-append
	InternalInconsistency Anomaly = 3  // internal-inconsistency
	IncompatibleOrder     Anomaly = 4  // incompatible-order
	AbortedRead           Anomaly = 5  // aborted-read
	IntermediateRead      Anomaly = 6  // intermediate-read
	G0                    Anomaly = 7  // G0
	G1c                   Anomaly = 8  // G1c
	GSingle               Anomaly = 9  // G-single
	GNonadjacent          Anomaly = 10 // G-nonadjacent
	G2                    Anomaly = 11 // G2
)

// Relation is a dependency of one committed transaction on another.
type Relation uint8

const (
	WW Relation = 1 // ww
	WR Relation = 2 // wr
	RW Relation = 3 // rw
)

// Transaction is a transaction that an anomaly involves.
type Transaction struct {
	Call       int   // the index of its invocation event
	Completion int   // the index of its completion event, and -1 for a pending transaction
	Kind       Kind  // OK, Fail or Unknown, and Invoke for a pending transaction
	Process    int
	Args       []any // its micro-operations, as its invocation states them
	Output     any   // its micro-operations with each read's list, for OK
}

// Link is one transaction of a cycle, with the relations of the edge to
// the next transaction.
type Link struct {
	Call      int
	Relations []Relation
}

// Evidence is one entry of an anomaly's explanation: an Edge of a cycle,
// or an Observation of any other anomaly.
type Evidence interface {
	json.Marshaler
	evidence()
}

// Edge is the evidence of one dependency of a cycle.
type Edge struct {
	From, To int // the calls of the two transactions
	Relation Relation
	Key      any
	Value    any  // see the table of the record
	Next     any  // see the table of the record
	Empty    bool // an rw edge whose From read the empty list
}

// Observation is the evidence of an anomaly that is no cycle.
type Observation struct {
	Anomaly   Anomaly
	Calls     []int   // the transaction that read, or the two of incompatible-order
	Key       any
	Reads     [][]any // the list that each of Calls read
	Value     any     // the value at fault
	Appender  int     // the call that appended Value
	Next      any     // the value that the appender appended after Value
	Expected  []any   // what the transaction knew of the list
	Whole     bool    // whether it knew the whole list, or only its end
	Future    any     // the transaction's later append that the read contains
	HasFuture bool    // whether the read contains one
}
```

The naming table fixes `Serializable` and `HasSnapshotIsolation`. This RFC
decides the other names:

| Name | Why Go needs it |
|---|---|
| `ErrTransaction` | A public package exports the sentinel of each fault that a caller tests |
| `Anomaly` and its eleven values | The `anomaly` and `kinds` fields state an enumeration, as the `outcome` field of `linearizable` states a `Verdict` |
| `Relation` and its three values | A link and an edge state their relations |
| `Transaction` | A transaction of the record states its completion kind. A `Span` states only whether its call is known, so an aborted transaction and one whose outcome is unknown would read alike |
| `Link` | The entries of the `cycle` field are values of one type |
| `Evidence`, `Edge`, `Observation` | The entries of the `explanation` field are of two kinds, and a sealed interface closes the set |
| `Edge.Empty`, `Observation.HasFuture` | An rw edge of an empty read states null for its value, and an internally inconsistent read without a later append states null for its future. Both differ from a value that is nil, whose typed literal is `{"type":"null"}` |

`Anomaly` and `Relation` are `uint8` enumerations with a `String` method
from stringer. Their line comments are the definition's spellings, and
each has a `Valid` method that `internal/enumtest` checks.

### The workload

A transaction is one call that the history recorded with the operation
`"txn"`. Its arguments are its micro-operations, each a `[]any` of three
elements: `"append"`, the key and the value, or `"read"`, the key and
nil. The `ok` output is a `[]any` that repeats the micro-operations, with
each read's list filled in, as any slice, or nil for the empty list:

```go
call := h.Invoke(client, "txn", []any{
	[]any{"append", "x", 7},
	[]any{"read", "y", nil},
}, "x", "y")
y, err := store.AppendThenRead(ctx, "x", 7, "y")
switch {
case errors.Is(err, ErrAborted):
	call.Fail(err)
case err != nil:
	call.Unknown(err)
default:
	call.OK([]any{
		[]any{"append", "x", 7},
		[]any{"read", "y", y},
	})
}
```

The checks identify a key, an appended value and each value of a read by
the text that `literal.Encode` writes for it. The history identifies its
keys the same way, so two values that one test treats as one key are one
key here.

A history that breaks the workload's contract ends the call with a fault
of the kind `ErrTransaction`, before the check derives anything. Its
operation is `history.Serializable` or `history.HasSnapshotIsolation`,
and its path names the call by the index of its invocation event:

| Rule | Path | Reason |
|---|---|---|
| The operation is not `"txn"` | `calls[4].operation` | `"read" is no transaction` |
| An argument is no micro-operation | `calls[4].args[1]` | `[]interface {}{"write", "x", 1} is no micro-operation` |
| A read states a list before it ran | `calls[4].args[1]` | `the read states a list before it ran` |
| The output is no list | `calls[4].output` | `the output "done" is no list of micro-operations` |
| The output does not repeat the micro-operations | `calls[4].output` | `the output repeats 1 of 2 micro-operations` |
| A micro-operation of the output differs | `calls[4].output[1]` | `[]interface {}{"append", "x", 8} does not repeat []interface {}{"append", "x", 7}` |
| A read returned no list | `calls[4].output[1]` | `the read returned 7, which is no list` |
| One call appends a value to one key twice | `calls[4].args[1]` | `7 is appended to "x" twice by call 4` |
| Two calls append a value to one key | `calls[6].args[0]` | `7 is appended to "x" by calls 4 and 6` |
| No typed literal states a key or a value | `calls[4].args[0]` | `chan int is no value that a typed literal states` |

A nil history is a fault without a kind, as it is for `Linearizable`.

### The derivation and the search

The checks follow the definition step by step, with these Go structures:

- **Transactions.** A slice in the order of the invocations. A
  transaction's number is its position in the slice, and the record names
  it by its call.
- **Identities.** A map from the text of each distinct typed literal to
  its number. A second map caches the number of a value of a predeclared
  type other than a float, so a value that recurs is encoded once. The
  derivation keeps the numbers of a read's values beside the values.
- **Appenders.** A map from the identities of a key and a value to the
  number of the transaction that appended the value. Each value of a
  committed read keeps the number of its appender, or -1 for none.
- **Version orders.** The longest committed read of each key, and a slice
  of the keys in the order the history first touches them. Of two reads
  of one list, the first counts.
- **Edges.** A map from a pair of transaction numbers to the position of
  the edge in a slice. An edge states its relations as a bit set and, for
  each relation, the position of its first evidence in derivation order.
  A cycle's explanation takes, of the relations that the search followed,
  the evidence that the derivation found first, as the reference does.
- **Adjacency.** The edges in compressed sparse row form: the edges that
  leave each transaction, in the order of their targets, each with its
  relations. A search over a set of relations skips the edges outside it.
- **Components.** Tarjan's algorithm without recursion, once for each set
  of relations that a search reads, so a long chain of transactions
  cannot grow the goroutine's stack without bound.
- **Paths.** A breadth-first search that visits each transaction's
  targets in ascending order. The `G-nonadjacent` search visits states of
  a transaction, whether a read-write edge led into it, and whether the
  walk has taken one, and an edge's relations in the order ww, wr, rw. A
  search marks the states that it visits with a number of its own in
  arrays that every search shares, so no search clears them.

The derivation, the six direct checks, the components and the searches
for `G0`, `G1c` and `G2` take time linear in the transactions and edges.
The searches for `G-single` and `G-nonadjacent` run one breadth-first
search per read-write edge of a component until one closes a cycle, which
is quadratic in the worst case. The checks do not call a function of the
caller, so they need no `recover`, and none exists.

### The record

Each check aborts only, as `Linearizable` does. A failing check reports
one record of its assertion, `serializable` or `snapshot-isolation`, with
the caller's contract and the five detail fields of the definition:

| Field | Go type |
|---|---|
| `anomaly` | `Anomaly`, the first kind in report order that the level forbids and the history exhibits |
| `kinds` | `[]Anomaly`, every kind that the level forbids and the history exhibits |
| `transactions` | `[]Transaction`, in the order the definition lists them for the anomaly |
| `cycle` | `[]Link` for a cycle, and nil for any other anomaly |
| `explanation` | `[]Evidence`: an `Edge` per link of a cycle, or one `Observation` |

The fields of an `Edge` and an `Observation` state what the definition's
explanation entries state:

| Entry | Fields that it states |
|---|---|
| A ww edge | `From`, `To`, `Relation`, `Key`, `Value`: a value of `From`, and `Next`: the value of `To` after it in the version order |
| A wr edge | `From`, `To`, `Relation`, `Key`, `Value`: the last value of the list that `To` read |
| An rw edge | `From`, `To`, `Relation`, `Key`, `Value`: the last value of the list that `From` read, `Empty` for the empty list, and `Next`: the value of `To` after it |
| `garbage-read`, `duplicate-append` | `Calls`, `Key`, `Reads` and `Value` |
| `internal-inconsistency` | `Calls`, `Key`, `Reads`, `Expected`, `Whole`, and `Future` with `HasFuture` |
| `incompatible-order` | `Calls` and `Reads` of two reads, and `Key` |
| `aborted-read` | `Calls`, `Key`, `Value` and `Appender` |
| `intermediate-read` | `Calls`, `Key`, `Value`, `Appender` and `Next` |

Every field that an entry does not state keeps its zero value. The JSON
form of each type, which `MarshalJSON` writes, states exactly the keys of
the definition's entry, each value as a typed literal, so a call record
states what the vector of the same history states.

`history` registers the sentences of `serializable` and
`snapshot-isolation` with `matcher.RegisterSentence` in its `init`
function, as it registers the sentence of `linearizable`. The text is
free under the definition. For a write skew, it reads:

```text
the store is serializable: G2
    kinds: G2
    call 0 -rw-> call 2: call 0 read [] from "x", and call 2 appended 2 to it
    call 2 -rw-> call 0: call 2 read [] from "y", and call 0 appended 1 to it
```

### Conformance

`conformance.VectorKind` gains `Serializable` and `SnapshotIsolation`,
and `conformance.Vectors` reads both files of `spec/corpus/history/`. The
runner records the vector's script through `New`, `Invoke`, `OK`, `Fail`
and `Unknown`, and runs the check of the vector's kind with an
`assert.Recorder`. A passing vector requires one call record of a pass. A
failing vector compares each field of the detail of the call record with
the vector's detail, with code of the runner's own.

### Testing

- Every source file has a black-box test file beside it, in the package
  `history_test`.
- The vectors pin the derivation, the six direct checks, the cycle search
  and its order, the reduction of a walk, and the record at both levels.
  A unit test pins each contract that the vectors do not state: each fault of the
  workload and its path, a key and a value of two Go types that one typed
  literal states, a pending transaction, a history that `FromIntervals`
  built, a nil history, the order of two components, the reduction of a
  walk that drops a part with adjacent read-write edges, the first
  evidence of a dependency that two keys reveal, the JSON form of each
  kind of evidence, and each sentence.
- Every exported function has a benchmark under
  `bench.Start(b).MaxAllocs(n)`. Its doc comment states the allocations.
- Two benchmarks check histories of 10,000 transactions that the test
  generates with a fixed seed, as the definition's measurements generated
  theirs: 8 clients over 16 keys, which retire after 32 appends, and up to
  4 micro-operations per transaction. One history comes from a store that
  runs each transaction whole at its commit, and one from a store that
  reads from the state at a transaction's start. A third benchmark checks
  the definition's constructed history of 10,001 transactions, on which
  the search for `G-single` is quadratic. The reference's times on the
  same histories are the baseline that the doc comments compare with.
- The coverage stage requires 100% statement coverage of `history`.
  gremlins measures `history` on demand against the bar of 100%. The
  mutation stage of `ergon check` covers the runners of `conformance`.

### Measurements

The mean of ten checks of each benchmark on one processor, against the
median of three checks of the reference on the same history, which the
benchmark's generator wrote out:

| History | Check | Go | Reference |
|---|---|---|---|
| Serializable store, 10,000 transactions | `Serializable`, passes | 29.5 ms | 0.490 s |
| Snapshot store, 10,000 transactions | `HasSnapshotIsolation`, passes | 20.7 ms | 0.444 s |
| Two chains, 10,001 transactions | `Serializable`, finds `G-nonadjacent` and `G2` | 134 ms | 2.221 s |

On these histories, Go checks 17 to 21 times as fast as the reference. On
the history of two chains, the search for `G-single` spends 85% of the
check, at about 4 ns per transaction that a breadth-first search visits.
A passing check of three transactions over one key allocates 89 times.
A passing check of a write skew allocates 86 times.

The Go checks were also compared with the reference on 2,640 histories of
10 to 400 transactions from three simulated stores, which serialize,
read a snapshot, or read the committed state, with aborts and unknown
outcomes. The detail of every failing check at both levels, 892 of
`serializable` and 521 of `snapshot-isolation`, was the reference's,
field for field, and every other check passed in both.

### Changes to the standards

`docs/standards/errors.md`, Writing: `history` registers the sentences of
`linearizable`, `serializable` and `snapshot-isolation` as `prop`
registers its sentences.

### The overlay

The overlay of definition 3.2.0 states no limit of the two checks and
declines no name. The limit on the history's recorder already covers the
histories that they read.

## Alternatives considered

### A. Values compared with `equality.Equal`

A key, an appended value and a value of a read compare as `assert.Equal`
compares them.

**Why not:** the definition identifies an append by the canonical text of
its key and its value. `equality.Equal` reports `int(1)` and `int64(1)` as
two values where the definition reports one. It also does not offer a
hash, so a map from an append to its appender would compare every pair of
values.

### B. A `Span` with a completion kind

`Span` gains a `Kind`, and the record's transactions are spans.

**Why not:** a span states a call as a model sees it, with an `Op` whose
`Known` field decides how the linearizability checker reads it. A kind
beside `Known` states the same fact twice, and the JSON form of a span
would change for every record of `linearizable`.

### C. The explanation as maps

The `explanation` field is a `[]map[string]any` with the keys of the
definition's entries.

**Why not:** every other record states Go values of declared types. A map
hides which keys an entry states, and a test that reads the evidence
would compare strings.

### D. A time limit

`Serializable` and `HasSnapshotIsolation` take the `TimeLimit` option of
`Linearizable`, and a check that passes it ends as undecided.

**Why not:** the definition rejects a budget for these searches, because
they are polynomial and always end. A limit would add an outcome that the
definition does not state.

### E. Maps keyed by calls and texts

The graph keys its transactions by their calls and its values by the
texts of their typed literals, in Go maps, as the reference keys its
dictionaries.

**Why not:** a breadth-first search then makes a map lookup per visited
transaction. On the three histories of the benchmarks, the checks took
225 ms, 134 ms and 2.03 s, 1.1 to 3.3 times as fast as the reference. The
encoding of every value of every read took 46% of the check of the
serializable store.

## Drawbacks

- **A check encodes each distinct key and value as a typed literal.** It
  encodes a value of a predeclared type once. It encodes any other value
  at each occurrence.
- **An `Observation` has fields that its anomaly does not use.** They keep
  their zero values. The doc comment lists which fields each anomaly
  states.
- **The searches for `G-single` and `G-nonadjacent` are quadratic in the
  worst case**, as the definition states.
- **`Evidence` is a sealed interface.** A caller reads an entry with a
  type switch over `Edge` and `Observation`.

## Open questions

None.

## Unresolved and future work

- Machines, which run the workload from two clients or more and call these
  checks in `settle`, are proposed in RFC-0011.

## References

| What | Where |
|---|---|
| Isolation checks over transaction histories | The definition's RFC-0015 |
| The executable reference and the 38 vectors, version 3.2.0 | <https://github.com/dokimasia/assert-spec>, `tools/history/isolation.py` and `corpus/history` |
| The history seam and the linearizability checker in Go, whose structure this RFC follows | RFC-0009 |
| Tarjan, "Depth-first search and linear graph algorithms", SIAM J. Comput. 1972 | <https://doi.org/10.1137/0201010> |
| Cerone and Gotsman, "Analysing Snapshot Isolation", PODC 2016 | <https://doi.org/10.1145/2933057.2933096> |
| Kingsbury and Alvaro, "Elle: Inferring Isolation Anomalies from Experimental Observations" | <https://arxiv.org/abs/2003.10554> |
