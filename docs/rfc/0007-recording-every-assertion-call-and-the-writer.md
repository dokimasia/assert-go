---
rfc: 0007
title: Recording every assertion call, and the writer
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-03
updated: 2026-10-04
discussion: none
supersedes: none
superseded-by: none
produces-adr: none
---

# RFC-0007: Recording every assertion call, and the writer

## Summary

Definition 2.3.0 adds RFC-0016 of the standard: a call record for every
assertion call, written into the artifact that the language's runner
writes. For Go, the artifact is the event stream of `go test -json`.
This RFC fixes how Go builds, numbers and writes call records, and how
one writer turns failure records and faults into the text that a seat
receives:

- Every assertion reports its verdict, `pass`, `fail` or `error`,
  through one of three functions of `internal/matcher`.
- A new package, `internal/record`, reads the switch, numbers the calls
  of a test and of a body, encodes a call record, and writes it through
  `Attr`.
- `assert.Recorder` keeps the call records of its calls, and `Records`
  returns them.
- The seat of a body, a property's case, an attempt of `Eventually` and
  the check of `Rejects`, hands its calls to the call that ran the body,
  with their run and phase.
- The faults that `prop` and `golden` send to a seat today, as `%v` or as
  a failure of `NoError`, end their call with the verdict `error`.
- A seat that implements `matcher.FaultReporter` receives a fault as the
  error it is, as a `Reporter` receives a failure's record. The module's
  tests compare records, faults and call records, and not the writer's
  text.
- The module vendors definition 2.3.0, and the conformance runners check
  the call record of every corpus case and the recording vectors.

The module's dependencies do not change.

## Motivation

The definition fixes the call record, the switch, the numbering and the
phases in `spec/recording.md`. The Go overlay declares the artifact: one
`attr` event per call record, under the key `dokimi.assert.<seq>`.

The standards of this module already describe the target, and the code
does not meet them:

| Rule of `docs/standards/errors.md` | Code today |
|---|---|
| Every assertion reports its verdict, pass or fail, through one function | An assertion calls `matcher.Fail` on a failure and nothing on a pass |
| Every text that the module sends to a seat comes from the writer | `prop` sends faults to `Fatalf` with `%v`, writes its notes as sentences, and renders its records with a renderer of its own beside `matcher.Render` |
| An error is a fault, and code never compares rendered text | `golden` reports an unreadable golden file as a failure of `NoError`, and the tests of the seats compare `Recorder.Message` |

Go decides what the definition leaves to a language:

- Where a call finds the place its record goes, for a `*testing.T`, a
  recorder, and the seat of a body.
- How the record of a call that runs a body takes its number before the
  calls of its body, when a property numbers the calls of a case only
  once it takes the case's result.
- How a run on more workers records only the calls that a run on one
  worker makes.
- How a Go value of any type becomes a typed literal.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| Switch | `internal/record` | Reads `DOKIMI_ASSERT_RECORD` once per process |
| Call | `internal/record` | The call record as a Go value, and its JSON line |
| Calls | `internal/record` | The calls of one test, of one recorder, or of one run of a body, in the order of their numbers |
| Attr sink | `internal/record` | Writes the call records of a test through `Attr`, split to test2json's line buffer |
| Typed literals | `internal/literal` | The codec of the definition's typed literals, moved from `internal/prop/literal`, with the opaque literal added |
| Verdict functions | `internal/matcher` | `Pass`, `Fail` and `Fault`: write the call record, then send a failure or a fault to the seat |
| Writer | `internal/matcher` | The text of a failure record and of a fault |
| Fault reporting | `internal/matcher` | `FaultReporter`, a seat that takes a fault as the error it is, and `NoteFault`, which notes a fault that does not end its call |
| Seats of a body | `internal/prop/engine`, `internal/matcher`, the root package | Keep the calls of one run, and hand them to the call that ran the body |
| `Recorder.Records` | the root package | Returns the encoded call records of the recorder's calls |
| Conformance | `conformance` | Checks the call record of every corpus case, and the recording vectors |

`internal/record` imports `internal/fault` and `internal/literal`, and
`internal/matcher` imports `internal/record`. The typed-literal codec
moves out of `prop` because every assertion now encodes its detail with
it, and a package of the property engine is no place for a codec of the
whole module.

### The switch

```go
// On reports whether a test's seat writes call records: false for an
// unset, empty or "0" DOKIMI_ASSERT_RECORD, and true for "1". It reads
// the variable once per process.
//
// # Errors
//
// Any other value is a fault of record.On at the variable, which every
// call of an assertion reports.
func On() (bool, error)
```

`On` reads the variable through `sync.OnceValues`. With recording off,
a call on a test's seat reads the cached result and does not build a
record. It allocates what it allocates today, and every allocation
ceiling of the module is unchanged.

`spec/recording.md` makes a recorder keep the records of its calls
whatever the switch states. The calls in a body are recorded when the
call that ran the body is recorded. The corpus runners read a
recorder's records without setting the variable.

### The call record

```go
// Call is the record of one assertion call, as the definition fixes it.
// The Calls that numbers the call sets its seq, parent, run and phase.
type Call struct {
	Assertion string
	Contract  string
	Verdict   Verdict // Pass, Fail or Error
	Aborting  bool
	Where     Where
	Detail    json.RawMessage
	Error     string
}
```

- `definition` is the constant `record.Definition`, `"2.3.0"`. A
  conformance test fails when it differs from the vendored `VERSION`.
- `where.file` is the base name of the frame that `matcher.CallerWhere`
  finds, the frame that a failure record states today.
- A call record is one line of JSON, with the fields in the order that
  the definition lists them, and without the fields that are not
  present.
- Go never writes `test`. Each `attr` event states its `Package` and its
  `Test`.

The detail of an assertion other than a property is an object of typed
literals. `literal.Detail` writes each value with `literal.Encode`, the
codec that a store's entry uses for a counterexample:

| Go value | Literal |
|---|---|
| `nil`, a nil pointer, and a nil interface | `null` |
| `bool`, every integer kind, every float kind, `string` | `bool`, `int`, `float`, `string` |
| A type that implements `encoding.TextMarshaler`, such as `time.Time` | `string`, its text |
| `[]byte` | `bytes` |
| A slice or an array | `list`, with `of` when every element is one scalar type and `items` otherwise |
| A map | `map`, its entries sorted by the JSON of their keys |
| A struct | `map` of its exported fields in declaration order |
| A pointer or an interface | The literal of the value it points to |
| An `error` and a `context.Context` | `opaque`, with the text that `%+v` writes |
| A value that `Encode` states no literal of: a `func`, a `chan`, a complex number, a value nested past 61 levels or of more than 65,536 parts | `opaque`, with the text that `%+v` writes |

A property's detail is the detail of its run, which `prop` encodes as
the form vectors write it: plain JSON, with the values of the draws as
typed literals. Each type of that detail, such as `prop.Drawn`,
implements `json.Marshaler` in that form.

### Numbering

`record.Calls` keeps the calls of one test, one recorder or one run of
a body. Each entry is a call's record, or the slot of a call that runs a
body and writes its record when it ends:

- The `Calls` of a test or of a recorder numbers a call when the call
  arrives: `seq` 1, 2 and on. A call that runs a body takes its number
  when it starts, so `parent` is lower than the `seq` of every call
  under it.
- The `Calls` of one run of a body keeps its calls in order and numbers
  none of them. When the call that ran the body takes the run's result,
  it moves the run's entries into its own `Calls`, each with `parent`,
  `run` and `phase`, and that `Calls` numbers them then.
- `parent` refers to the entry of the call that ran the body, and takes
  that entry's number when the entry is numbered. A body inside a body,
  such as an `Eventually` in a property's case, numbers correctly.

A test's `Calls` writes each record through the test's `Attr` once the
record is complete. A recorder's `Calls` keeps the encoded records for
`Records`.

### The `Calls` of a seat

The verdict functions receive a `matcher.Seat`, which declares
`Helper`, `Fatalf` and `Errorf`. `internal/record` finds the `Calls` of
a seat in this order:

1. A seat of this module, `assert.Recorder`, `engine.Case` and the
   seats of `Eventually` and `Rejects`, embeds `record.Calls` through an
   unexported alias. `record` finds it through an interface whose one
   method is unexported. A type in another package satisfies such an
   interface through the promoted method, and `go doc` lists neither
   the field nor the method. We tested both on Go 1.27.1 before writing
   this RFC.
2. `*testing.T`, `*testing.B` and `*testing.F` declare `Attr` and
   `Name`. `record` keeps a `Calls` for each of them in a `sync.Map`
   keyed by the test. Each subtest is a test of its own, so its calls
   are numbered from 1. An entry is kept for the life of the process,
   because a cleanup of the test can make calls until the test ends. A
   cleanup that removed the entry earlier would number the later calls
   from 1 again. An entry contains a counter and the test's seat, so the
   process keeps the seat of every test that records a call until it
   exits.
3. Any other seat does not write call records, because it has no
   artifact.

The module seals `FormOption` with `matcher.FormSeal`, an exported
method whose parameter no other module can name. The seats are public
types whose `go doc` a caller reads, so they use the embedding, which
does not add an exported name.

### The `Attr` sink

- Each record is one `Attr` under the key `dokimi.assert.<seq>`, with
  the record's JSON as its value.
- `testing` writes `=== ATTR  <test> <key> <value>` under `-v` and
  `-json` alone, after the framing byte `^V` under `-json`. test2json
  turns a line into an `attr` event only when the whole line, its
  newline included, fits its 4,096-byte buffer (Go 1.27.1,
  `src/cmd/internal/test2json/test2json.go`, `inBuffer`).
- A record whose line would not fit is split over consecutive `Attr`
  calls of the same key, at UTF-8 boundaries, each line within the
  buffer. A reader joins the values of one key in order.
- JSON escapes every control character, so a value never contains the
  newline that `Attr` refuses.

### The verdict functions

```go
// Pass reports that a call of assertion passed.
func Pass(seat Seat, mode Mode, assertion, contract string)

// Fail reports that a call failed with detail, which contains exactly
// the fields that the assertion declares.
func Fail(seat Seat, mode Mode, assertion, contract string, detail map[string]any)

// Fault reports that a call ended without a verdict, because err refused
// an argument or the environment.
func Fault(seat Seat, mode Mode, assertion, contract string, err error)

// FaultReporter is a Seat that takes a fault as the error it is, as a
// Reporter takes a failure's record. ending is true for the fault of a
// call that ends without a verdict, and false for a fault that NoteFault
// notes while its call runs on.
type FaultReporter interface {
	ReportFault(err error, ending bool)
}
```

- Each writes the call record first, when the seat's `Calls` records,
  and then reports. `Pass` then returns. `Fail` sends the record to a
  `Reporter`, and the writer's text to any other seat, through `Fatalf`
  under `Fatal` and `Errorf` under `Soft`, as `Fail` does today.
- `Fault` passes `err` to a `FaultReporter` with `ending` true, and
  sends the writer's text of `err` to any other seat through `Fatalf`.
  It does so under both modes: the call ended without a verdict, and a
  test that ran on would report the same fault again.
- `Pass` sends the seat nothing, so it marks no frame as a helper.
  `testing`'s `Helper` reads the caller's stack on every call.
- An invalid switch makes every call, `Pass` included, end with the
  switch's fault as `Fault` ends a call, and write no record. The call
  marks its frames as helpers first, so the fault names the caller's
  line.
- Every assertion of `internal/matcher` calls `Pass` on its passing
  path. `assert` and `expect` do not change: they call the matcher.
- A call that runs a body opens its entry with `record.Begin` before the
  first run, hands each run's calls to the entry, and closes the entry
  through the same three functions.

### Calls in a body

| Assertion | Seat of the body | Run | Phase |
|---|---|---|---|
| `prop-for-all` and the forms | `engine.Case` | Every case that the runner takes, in the order of a run on one worker | The case's phase |
| `eventually` | The seat of the attempt | Every attempt | None |
| `rejects` | The seat of the check | 1 | None |

The engine states the phase of every execution: `example` for the case
of `Draws` and every case of `Example`, then `stored`, `simplest`,
`random`, `prefix`, `edge`, `coverage`, `replay`, `shrink`, `explain`,
`token` for `RunReplay`, and `fuzz` for `Bridge`. It hands a case's
calls to the property's entry where it takes the case's result:

- On one worker, a case that repeats a tested case or diverges stops at
  that choice, so its body does not call an assertion after it.
- On more workers, a case ahead runs outside the case tree. Each call
  that it keeps states the number of steps that its walk had made.
  `entered` returns the step at which a run on one worker stops the
  case, and the case hands over only the calls made before that step.
- A case ahead that the runner never takes hands over nothing, and
  neither do the shrink candidates after the one that a batch accepts,
  whose runs the budget does not count.
- A property's own record is written when its run ends, in the slot it
  opened, with its run's detail on a pass too.

A fuzz input runs as a test of its own, so each input writes one
`prop-for-all` record, and the calls of its case under the phase `fuzz`.

### `Recorder.Records`

```go
// Records returns the call record of every call that the recorder
// received, one JSON line each, in the order of their numbers. A call in
// a body that ran on the recorder, such as a case of prop.ForAll, is
// among them. The slice is a fresh copy.
func (r *Recorder) Records() []string
```

The recorder numbers its calls from 1 and writes none of them to a
test's output. `Failures`, `Message` and `Messages` keep their meaning.

### The writer

```go
// Writer turns the records and the faults of this module into text.
type Writer interface {
	// Failure returns the text of a failure's record.
	Failure(f Failure) string
	// Fault returns the text of a fault.
	Fault(err error) string
}
```

- The text writer is the one implementation. `Render` returns its text
  of a failure's record, and `RenderFault` its text of a fault, which is
  the fault's `Error`.
- The sentence of a property's record is `prop`'s: the outcome with the
  counts, a line per draw, the failing case's record, how to replay it,
  the other failures, and the divergence or the shortfall. `matcher`
  cannot import `prop`, so `prop` registers that sentence for its 39
  assertions in `init`, in a table of the writer that nothing writes
  after `init`. The initialisation of a package completes before any
  test starts, so the table does not need a lock.
- Every text that the module sends to a seat comes from the writer:
  failures, faults, and the notes that `prop` logs. `Note` writes the
  note of a failing case into the log of a seat that has `Logf`.
- A fault of the store that does not end the run, such as a damaged
  file that the run skips, goes to `NoteFault`. A `FaultReporter`
  receives it as the error it is, with `ending` false. Any other seat
  with a log receives the writer's text of it.
- Tests compare the values that the module reports: a failure's record,
  a fault, and a call record. The seat of `internal/matchertest`
  implements `Reporter` and `FaultReporter`, so a test reads each
  failure as its record and each fault as the error it is.
- A call record states the writer's text of a fault in its field
  `error`. A test builds the text that it expects with `RenderFault`,
  from the fault that it expects.
- A test compares the writer's text only where no value states it: in a
  test of the writer or of a registered sentence, and in a test of what
  a seat without `Report` and `ReportFault` receives, such as the log of
  a fuzz input's test.

A second writer, such as one for JSON, is out of this RFC. The call
records are the machine-readable form of every call, and no caller has
asked for failure text in another form. A caller that needs one would
reverse this.

### Faults that end a call

| Call | Today | With this RFC |
|---|---|---|
| `prop.ForAll`, `prop.Fuzz` and the forms: profile, seed, token, `Draws`, store, a second property of one contract, a refused draw | `Fatalf("%v", err)` | `Fault`, with the verdict `error` |
| `golden`: a file that cannot be read or written, a value that is no JSON | A failure of `err-absent` | `Fault` of the golden assertion |
| Every assertion: an invalid switch | Not possible | `Fault`, without a call record |

### Conformance

- `Case.Check` also reads the recorder's one call record: `seq` 1
  without a `parent`, the case's assertion, the contract unchanged, the
  verdict that `expect` states, `aborting` for the surface the runner
  used, and the detail decoded from its typed literals and compared with
  the case's detail. A passing case's record states no detail.
- The recording vectors run each body under `prop.ForAll` on a
  recorder. The example of a vector becomes a `Draws` entry, the literal
  that the vector's choices decode to. The runner compares the run,
  phase and verdict of each call in order, `seq` k + 1 and `parent` 1 of
  the call at position k, and the property's verdict.
- A behaviour vector's passing run is compared through the detail of the
  property's call record, which replaces the second run of the engine
  that the runner makes today.
- `make spec-sync` vendors definition 2.3.0. The completeness gate then
  requires `Recorder.Records`.

### Failure handling

| Condition | Behaviour |
|---|---|
| `DOKIMI_ASSERT_RECORD` is neither empty, `0` nor `1` | Every call ends with the switch's fault, as `Fault` ends a call, and writes no record |
| A detail value that no typed literal states | An opaque literal of its `%+v` text |
| A value nested past 61 levels or of more than 65,536 parts, such as a cyclic structure | An opaque literal of the whole value |
| A record longer than one test2json line | Split over `Attr` calls of one key |
| A call on a seat without an artifact | No record, and the failure is reported as today |
| A call from a goroutine that outlived its test | `testing` panics on its failure, as today. A pass writes its record with the test's next number |
| A fault of the store that does not end the run, such as a damaged file | `NoteFault` passes it to a `FaultReporter`, and writes its text into the log of any other seat |

### Allocation contract

- With recording off, a call on a test's seat allocates what it
  allocated before this RFC. A benchstat comparison with that code finds
  the same allocations on both sides: for passing calls of `Equal`,
  `True`, `Contains`, `NoError` and `ErrorIs`, for failing calls of
  `Equal`, `True` and `Contains`, and for `expect.Equal` either way.
- A recorded call allocates its record and the encoding of its detail.
- The benchmarks and the `Test<Subject>Allocs` tests measure on a seat
  that writes no call record, so recording changes no ceiling of the
  module.

### Standards

- `docs/standards/errors.md` states that a recorder keeps its calls'
  records whatever the switch states, that the allocation ceilings
  apply to a call on a seat that writes no call record, and that a
  `FaultReporter` receives a fault as the error it is.
- `docs/standards/testing.md` states that a test compares records,
  faults and call records. It compares the writer's text only where no
  value states it: in a test of the writer or of a registered sentence,
  and in a test of what a seat without `Report` and `ReportFault`
  receives.

## Alternatives considered

### A. Run the conformance runners in a second process with the switch on

**Why not:**

- The coverage of a second process is missing from the profile of
  `go test -cover` without more machinery.
- Every language would need the same machinery.
- A recorder's records are the outcome that a test reads, and not output
  of the run.

### B. An exported method that returns a seat's `Calls`

`Recorder`, `prop.Case` and the other seats declare `Calls() *record.Calls`.

**Why not:** it adds a name that the naming table does not state, and
`go doc` lists a method that no caller can use. The embedding adds no
name.

### C. A registry of every seat

`record` keeps a `Calls` for every seat in a map, as it does for a
`*testing.T`.

**Why not:** a recorder and a property's case have no `Cleanup` that
would remove their entry, so the map would grow by one entry for every
case of every property.

### D. A property's detail as typed literals

**Why not:**

- The run fixes the type of every field, so a type tag adds no fact.
- The failure record in the detail would become a `record` literal, a
  form in which no reader of a run expects a failure record.
- The definition states the property's detail in the form of the form
  vectors.

### E. A `Writer` that a caller selects

`assert.WithWriter(w)` replaces the text writer for the process.

**Why not:** no second writer exists, and the call records already give
a program every fact of a call.

### F. Tests read a fault as the writer's text

A test reads the text that a seat's `Fatalf` or log received, such as
`Recorder.Message`, and compares it with the text that it expects.

**Why not:**

- A second writer would change that text, and every such test with it.
- `docs/standards/errors.md` forbids comparing the text that `Error`
  renders.
- `Reporter` already passes a failure's record. A fault takes the same
  path through `FaultReporter`, at the cost of one interface.

## Drawbacks

- Every passing call reads the switch and asserts the type of its seat.
  Against the code before this RFC, on one core with recording off, a
  passing `True`, `Contains`, `NoError` or `ErrorIs` takes 11 to 14 ns
  more, 7 to 9% of its 152 to 158 ns. A passing `Equal` takes 1.5 µs,
  and benchstat finds no change in its time (p = 0.97). The figures are
  benchstat's medians of 12 runs per side, in alternating rounds.
- A failing call takes 2 to 20% longer, 60 to 240 ns, and allocates the
  same.
- A recorder builds a call record for every call it receives, so a test
  that drives many calls on a recorder allocates more than today.
- The sentence of a property's record is registered in a table at
  `init`, so the writer depends on `prop` having been initialised for
  the text of the 39 property assertions.
- `golden` ends the call of an unreadable golden file with a fault
  instead of a failure of `NoError`, so a test that reads that failure's
  record changes.
- A run on more workers keeps, with every call of a case ahead, the
  number of steps of its walk.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| The call record, the switch, the numbering and the phases | Definition 2.3.0, `spec/recording.md` |
| The opaque literal | Definition 2.3.0, `spec/encoding.md` |
| The Go artifact | Definition 2.3.0, `overlays/go.json`, `records` |
| `testing.common.Attr` | Go 1.27.1, `src/testing/testing.go`, line 1757 |
| test2json's line buffer | Go 1.27.1, `src/cmd/internal/test2json/test2json.go`, `inBuffer` and `lineBuffer` |
| The rules this RFC meets | `docs/standards/errors.md`, `testing.md` and `code.md` |
| The engine's cases ahead | RFC-0002, and `internal/prop/engine/ahead.go` |
